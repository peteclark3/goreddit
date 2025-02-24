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

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			var post reddit.Post
			if err := json.Unmarshal(message.Value, &post); err != nil {
				log.Printf("Error unmarshaling post: %v", err)
				continue
			}

			if err := c.store.SavePost(ctx, post); err != nil {
				log.Printf("Error saving post: %v", err)
				continue
			}

			log.Printf("Saved post: %s from r/%s", post.Title, post.Subreddit)
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
	defer close(posts)

	log.Printf("Starting consumer with brokers: %v, topic: %s", c.cfg.Kafka.Brokers, c.cfg.Kafka.Topic)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			log.Printf("Received message from Kafka: %s", string(message.Value))

			var post reddit.Post
			if err := json.Unmarshal(message.Value, &post); err != nil {
				log.Printf("Error unmarshaling post: %v", err)
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

			// Save to DB if store is provided
			if c.store != nil {
				if err := c.store.SavePost(ctx, post); err != nil {
					log.Printf("Error saving post: %v", err)
					// Continue anyway to send to WebSocket
				}
			}

			// Send to WebSocket channel
			posts <- post
		}
	}
}
