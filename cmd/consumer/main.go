package main

import (
	"context"
	"goreddit/internal/config"
	"goreddit/internal/kafka"
	"goreddit/internal/storage"
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

	// Create PostgreSQL store
	store, err := storage.NewPostgresStore(cfg)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(cfg, store)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consumer in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Shutting down...")
		cancel()
	case err := <-errCh:
		log.Printf("Consumer error: %v", err)
		cancel()
	}

	// Wait for consumer to finish
	if err := <-errCh; err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
} 