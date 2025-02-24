package reddit

import (
	"context"
	"goreddit/internal/config"
	"testing"
	"time"
)

func TestRedditClient(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create Reddit client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	posts := make(PostChannel, 100)
	go client.StreamPosts(ctx, posts)

	// Check if we receive any posts
	select {
	case post := <-posts:
		if post.ID == "" {
			t.Error("Received empty post ID")
		}
		if post.Title == "" {
			t.Error("Received empty post title")
		}
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for posts")
	}
} 