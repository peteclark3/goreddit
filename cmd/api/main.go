package main

import (
	"goreddit/internal/api"
	"goreddit/internal/config"
	"log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	server := api.NewServer(cfg)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
