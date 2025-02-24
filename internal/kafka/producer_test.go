package kafka

import (
	"context"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"testing"
	"time"
)

func TestKafkaProducer(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Create a test post
	post := reddit.Post{
		ID:        "test123",
		Title:     "Test Post",
		Body:      "Test Body",
		Subreddit: "test",
		Score:     100,
		URL:       "https://reddit.com/test",
		CreatedAt: float64(time.Now().Unix()),
	}

	// Send post to Kafka
	ctx := context.Background()
	posts := make(reddit.PostChannel, 1)
	posts <- post

	// Start producer in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- producer.Start(ctx, posts)
	}()

	// Wait for potential errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Producer failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		// Test passed
	}
} 