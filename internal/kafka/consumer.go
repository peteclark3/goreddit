package kafka

import (
	"context"
	"encoding/json"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"goreddit/internal/storage"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/cdipaolo/sentiment"
	kafka "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	store  *storage.PostgresStore
	cfg    *config.Config
}

func NewConsumer(cfg *config.Config, store *storage.PostgresStore) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})

	return &Consumer{
		reader: reader,
		store:  store,
		cfg:    cfg,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	defer c.reader.Close()

	log.Printf("STANDALONE CONSUMER: Starting with brokers: %v, topic: %s", c.cfg.Kafka.Brokers, c.cfg.Kafka.Topic)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("STANDALONE CONSUMER: Error reading message: %v", err)
				continue
			}

			var post reddit.Post
			if err := json.Unmarshal(message.Value, &post); err != nil {
				log.Printf("STANDALONE CONSUMER: Error unmarshaling post: %v", err)
				continue
			}

			if err := c.store.SavePost(ctx, post); err != nil {
				log.Printf("STANDALONE CONSUMER: Error saving post: %v", err)
				continue
			}

			log.Printf("STANDALONE CONSUMER: Saved post: %s from r/%s", post.Title, post.Subreddit)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) extractTopics(text string) []string {
	// Convert to lowercase and remove special characters
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[^a-zA-Z\s]`)
	text = reg.ReplaceAllString(text, " ")

	// Split into words
	words := strings.Fields(text)

	// Filter common words and short words
	topics := make(map[string]bool)
	stopWords := map[string]bool{
		"the": true, "be": true, "to": true, "of": true, "and": true,
		"a": true, "in": true, "that": true, "have": true, "i": true,
		"it": true, "for": true, "not": true, "on": true, "with": true,
		"he": true, "as": true, "you": true, "do": true, "at": true,
		"this": true, "but": true, "his": true, "by": true, "from": true,
		"they": true, "we": true, "say": true, "her": true, "she": true,
		"or": true, "an": true, "will": true, "my": true, "one": true,
		"all": true, "would": true, "there": true, "their": true,
		"reddit": true, "post": true, "comment": true, "thread": true,
		"just": true, "like": true, "want": true, "need": true, "got": true,
		"see": true, "know": true, "think": true, "way": true, "time": true,
		"people": true, "other": true, "same": true, "good": true, "bad": true,
		"great": true,
	}

	for _, word := range words {
		// Skip stop words, short words, and common Reddit terms
		if !stopWords[word] && len(word) > 3 {
			// Capitalize first letter for better presentation
			word = strings.Title(strings.ToLower(word))
			topics[word] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(topics))
	for topic := range topics {
		result = append(result, topic)
	}

	return result
}

func (c *Consumer) analyzeSentiment(text string) float64 {
	// Initialize sentiment analyzer
	model, err := sentiment.Restore()
	if err != nil {
		log.Printf("Error initializing sentiment analyzer: %v", err)
		return 0
	}

	// Combine title and body for analysis
	analysis := model.SentimentAnalysis(strings.ToLower(text), sentiment.English)

	// Convert 0/1 score to -1 to 1 range and handle neutral case
	if analysis.Score == 0 {
		return -1.0 // Negative
	}
	return 1.0 // Positive
}

func (c *Consumer) StartWithChannel(ctx context.Context, posts chan<- reddit.Post) error {
	defer c.reader.Close()
	defer func() {
		log.Printf("API CONSUMER: Closing posts channel")
		close(posts)
	}()

	log.Printf("API CONSUMER: Starting with brokers: %v, topic: %s", c.cfg.Kafka.Brokers, c.cfg.Kafka.Topic)
	messageCount := 0
	lastLogTime := time.Now()
	lastWaitingLog := time.Now()
	waitingMessageShown := false

	for {
		select {
		case <-ctx.Done():
			log.Printf("API CONSUMER: Context cancelled, shutting down")
			return nil
		default:
			// Log periodic status
			now := time.Now()
			if now.Sub(lastLogTime) >= time.Second*10 {
				if messageCount == 0 {
					log.Printf("API CONSUMER: No messages received in last 10 seconds. Still waiting...")
				} else {
					log.Printf("API CONSUMER: Status update - Processed %d messages in last 10 seconds", messageCount)
				}
				messageCount = 0
				lastLogTime = now
			}

			// Show "waiting" message only once every 30 seconds
			if !waitingMessageShown || now.Sub(lastWaitingLog) >= time.Second*30 {
				log.Printf("API CONSUMER: [%v] Waiting for messages from Kafka topic '%s'...", time.Now().Format("15:04:05.000"), c.cfg.Kafka.Topic)
				waitingMessageShown = true
				lastWaitingLog = now
			}

			readStart := time.Now()
			message, err := c.reader.ReadMessage(ctx)
			readDuration := time.Since(readStart)

			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					log.Printf("API CONSUMER: Error reading message: %v", err)
				}
				time.Sleep(time.Second) // Add small delay on errors
				continue
			}

			// Reset waiting message flag when we get a message
			waitingMessageShown = false
			messageCount++

			processStart := time.Now()
			log.Printf("API CONSUMER: [%v] 📫 RECEIVED MESSAGE 📫 - Key: %s (read took %v)",
				processStart.Format("15:04:05.000"),
				string(message.Key),
				readDuration)

			var post reddit.Post
			if err := json.Unmarshal(message.Value, &post); err != nil {
				log.Printf("API CONSUMER: Error unmarshaling post: %v", err)
				continue
			}

			// Extract topics from title and body
			text := post.Title
			if post.Body != "" {
				text += " " + post.Body
			}
			post.Topics = c.extractTopics(text)

			// Analyze sentiment
			post.Sentiment = c.analyzeSentiment(text)

			processDuration := time.Since(processStart)
			log.Printf("API CONSUMER: [%v] Processed post in %v - ID: %s, Title: %s",
				time.Now().Format("15:04:05.000"),
				processDuration,
				post.ID,
				post.Title)

			// Save to DB if store is provided
			if c.store != nil {
				dbStart := time.Now()
				if err := c.store.SavePost(ctx, post); err != nil {
					log.Printf("API CONSUMER: Error saving post: %v", err)
					// Continue anyway to send to WebSocket
				} else {
					log.Printf("API CONSUMER: Saved to DB in %v", time.Since(dbStart))
				}
			}

			// Send to WebSocket channel - use blocking send
			sendStart := time.Now()
			log.Printf("API CONSUMER: [%v] Attempting to send to API channel - ID: %s",
				sendStart.Format("15:04:05.000"),
				post.ID)

			select {
			case posts <- post:
				sendDuration := time.Since(sendStart)
				log.Printf("API CONSUMER: [%v] ✅ Sent to API channel in %v - ID: %s",
					time.Now().Format("15:04:05.000"),
					sendDuration,
					post.ID)
			case <-ctx.Done():
				log.Printf("API CONSUMER: Context cancelled while trying to send post - ID: %s", post.ID)
				return nil
			}
		}
	}
}
