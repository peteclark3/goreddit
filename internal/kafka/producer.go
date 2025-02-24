package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"log"
	"strings"

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

	// Create topic if it doesn't exist
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             cfg.Kafka.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

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
	for {
		select {
		case <-ctx.Done():
			return p.writer.Close()
		case post := <-posts:
			if err := p.sendPost(ctx, post); err != nil {
				log.Printf("Error sending post to Kafka: %v", err)
				continue
			}
		}
	}
}

// sendPost serializes and sends a single post to Kafka
func (p *Producer) sendPost(ctx context.Context, post reddit.Post) error {
	value, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal post: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(post.ID),
		Value: value,
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the Kafka writer
func (p *Producer) Close() error {
	return p.writer.Close()
}
