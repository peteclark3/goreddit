package api

import (
	"context"
	"encoding/json"
	"fmt"
	"goreddit/internal/config"
	"goreddit/internal/kafka"
	"goreddit/internal/reddit"
	"goreddit/internal/storage"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	cfg      *config.Config
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mutex    sync.Mutex
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) Start() error {
	// Create store for persistence
	store, err := storage.NewPostgresStore(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	defer store.Close()

	// Create Kafka consumer with store
	consumer, err := kafka.NewConsumer(s.cfg, store) // Pass the store
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	defer consumer.Close()

	// Handle WebSocket connections
	http.HandleFunc("/ws", s.handleWebSocket)

	// Serve static files for Vue.js frontend
	http.Handle("/", http.FileServer(http.Dir("./frontend/dist")))

	// Start consuming posts in background
	go s.consumePosts(consumer)

	log.Printf("Starting WebSocket server on port %d", s.cfg.API.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.API.Port), nil)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("New WebSocket connection attempt from %s", r.RemoteAddr)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket client connected from %s", conn.RemoteAddr())

	s.mutex.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.mutex.Unlock()

	log.Printf("Total connected clients: %d", clientCount)

	// Remove client when connection closes
	defer func() {
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()
		log.Printf("Client disconnected, remaining clients: %d", len(s.clients))
	}()

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading from client: %v", err)
			break
		}
	}
}

func (s *Server) consumePosts(consumer *kafka.Consumer) {
	ctx := context.Background()
	posts := make(chan reddit.Post)

	// Start consuming in background
	go func() {
		log.Printf("Starting Kafka consumer for WebSocket broadcast")
		if err := consumer.StartWithChannel(ctx, posts); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Broadcast posts to all WebSocket clients
	for post := range posts {
		log.Printf("Broadcasting post to %d clients: %s", len(s.clients), post.Title)
		s.broadcastPost(post)
	}
}

func (s *Server) broadcastPost(post reddit.Post) {
	data, err := json.Marshal(post)
	if err != nil {
		log.Printf("Error marshaling post: %v", err)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.Printf("Broadcasting post to %d clients. Topics: %v", len(s.clients), post.Topics)

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Error sending to client: %v", err)
			client.Close()
			delete(s.clients, client)
			continue
		}
		log.Printf("Successfully sent post to client")
	}
}
