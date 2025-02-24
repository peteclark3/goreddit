package storage

import (
	"context"
	"goreddit/internal/config"
	"goreddit/internal/reddit"
	"testing"
	"time"
)

func TestPostgresStore(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	store, err := NewPostgresStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Test saving a post
	post := reddit.Post{
		ID:        "test_" + time.Now().String(),
		Title:     "Test Post",
		Body:      "Test Body",
		Subreddit: "test",
		Score:     100,
		URL:       "https://reddit.com/test",
		CreatedAt: float64(time.Now().Unix()),
	}

	err = store.SavePost(context.Background(), post)
	if err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	// Test updating the same post
	post.Score = 200
	err = store.SavePost(context.Background(), post)
	if err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}
} 