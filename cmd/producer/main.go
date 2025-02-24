package main

import (
	"context"
	"goreddit/internal/config"
	"goreddit/internal/kafka"
	"goreddit/internal/reddit"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Reddit client
	redditClient, err := reddit.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Reddit client: %v", err)
	}

	// Create Kafka producer
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel for Reddit posts
	posts := make(reddit.PostChannel, 100)

	// Start Reddit post stream
	go redditClient.StreamPosts(ctx, posts)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start producer in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- producer.Start(ctx, posts)
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Shutting down...")
		cancel()
	case err := <-errCh:
		log.Printf("Producer error: %v", err)
		cancel()
	}

	// Wait for producer to finish
	if err := <-errCh; err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
} 