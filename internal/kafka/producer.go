package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"log"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	cfg    *config.Config
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.Config) (*Producer, error) {
	// Create admin client to manage topics
	conn, err := kafka.Dial("tcp", cfg.Kafka.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to dial kafka: %w", err)
	}
	defer conn.Close()

	// Delete topic if it exists
	log.Printf("Attempting to delete topic %s if it exists...", cfg.Kafka.Topic)
	err = conn.DeleteTopics(cfg.Kafka.Topic)
	if err != nil && !strings.Contains(err.Error(), "unknown topic") {
		log.Printf("Warning: Failed to delete topic: %v", err)
	} else {
		log.Printf("Successfully deleted topic %s", cfg.Kafka.Topic)
	}

	// Wait a moment for deletion to complete
	time.Sleep(2 * time.Second)

	// Create topic
	log.Printf("Creating topic %s...", cfg.Kafka.Topic)
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             cfg.Kafka.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}
	log.Printf("Successfully created topic %s", cfg.Kafka.Topic)

	// Create the writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
		cfg:    cfg,
	}, nil
}

// Start begins consuming from the Reddit client and producing to Kafka
func (p *Producer) Start(ctx context.Context, posts reddit.PostChannel) error {
	// Create a map for O(1) lookup of valid subreddits
	validSubreddits := make(map[string]bool)
	for _, sub := range reddit.TargetSubreddits {
		validSubreddits[strings.ToLower(sub)] = true
	}

	log.Printf("Producer: Starting with target subreddits: %s", strings.Join(reddit.TargetSubreddits, ", "))

	for {
		select {
		case <-ctx.Done():
			return p.writer.Close()
		case post := <-posts:
			// Skip posts from non-target subreddits
			if !validSubreddits[strings.ToLower(post.Subreddit)] {
				log.Printf("Producer: Skipping post from non-target subreddit: r/%s", post.Subreddit)
				continue
			}

			if err := p.sendPost(ctx, post); err != nil {
				log.Printf("Error sending post to Kafka: %v", err)
				continue
			}
		}
	}
}

// sendPost serializes and sends a single post to Kafka
func (p *Producer) sendPost(ctx context.Context, post reddit.Post) error {
	log.Printf("Producer: Sending post to Kafka - ID: %s, Title: %s, Subreddit: %s", post.ID, post.Title, post.Subreddit)

	value, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal post: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(post.ID),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	log.Printf("Producer: Successfully sent post to Kafka - ID: %s", post.ID)
	return nil
}

// Close closes the Kafka writer
func (p *Producer) Close() error {
	return p.writer.Close()
}
