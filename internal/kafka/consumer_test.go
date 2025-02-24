package kafka

import (
	"context"
	"encoding/json"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"goreddit/internal/storage"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func TestConsumer(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create store
	store, err := storage.NewPostgresStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create producer to send test messages
	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Create consumer
	consumer, err := NewConsumer(cfg, store)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Create test post
	post := reddit.Post{
		ID:        "test_" + time.Now().String(),
		Title:     "Test Consumer Post",
		Body:      "Test Consumer Body",
		Subreddit: "test",
		Score:     100,
		URL:       "https://reddit.com/test",
		CreatedAt: float64(time.Now().Unix()),
	}

	// Send post to Kafka
	ctx := context.Background()
	value, _ := json.Marshal(post)
	err = producer.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(post.ID),
		Value: value,
	})
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Start consumer in goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(ctx)
	}()

	// Wait for potential errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Consumer failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		// Test passed
	}
}
