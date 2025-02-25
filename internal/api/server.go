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
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	cfg      *config.Config
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mutex    sync.Mutex
	// Add a buffer of recent posts
	recentPosts []reddit.Post
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients:     make(map[*websocket.Conn]bool),
		recentPosts: make([]reddit.Post, 0, 100), // Keep last 100 posts
	}
}

func (s *Server) Start() error {
	log.Printf("Starting API server, clearing recent posts buffer...")
	s.recentPosts = make([]reddit.Post, 0, 100) // Reset recent posts buffer

	// Create store for persistence
	store, err := storage.NewPostgresStore(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	defer store.Close()

	// Create Kafka consumer with store and unique group ID
	apiConfig := *s.cfg                                    // Make a copy of the config
	apiConfig.Kafka.GroupID = s.cfg.Kafka.GroupID + "-api" // Add suffix for API consumer

	consumer, err := kafka.NewConsumer(&apiConfig, store)
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

	log.Printf("WebSocket client connected from %s", conn.RemoteAddr())

	// Add client to the pool
	s.mutex.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)

	// Send recent posts to the new client
	if len(s.recentPosts) > 0 {
		log.Printf("Sending %d recent posts to new client", len(s.recentPosts))
		for _, post := range s.recentPosts {
			data, err := json.Marshal(post)
			if err != nil {
				log.Printf("Error marshaling post for new client: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Error sending recent post to new client: %v", err)
				break
			}
			// Small delay to prevent overwhelming the client
			time.Sleep(50 * time.Millisecond)
		}
	}
	s.mutex.Unlock()

	log.Printf("Total connected clients: %d", clientCount)

	// Remove client when connection closes
	defer func() {
		s.mutex.Lock()
		delete(s.clients, conn)
		conn.Close()
		s.mutex.Unlock()
		log.Printf("Client disconnected, remaining clients: %d", len(s.clients))
	}()

	// Keep connection alive and handle client messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (s *Server) consumePosts(consumer *kafka.Consumer) {
	log.Printf("API: consumePosts function started")

	ctx := context.Background()
	posts := make(chan reddit.Post, 100) // Buffered channel to prevent blocking

	log.Printf("API: Created posts channel with buffer size 100")

	messageCount := 0
	lastLogTime := time.Now()

	// Start consuming in background
	go func() {
		log.Printf("API: Starting Kafka consumer goroutine for WebSocket broadcast")
		if err := consumer.StartWithChannel(ctx, posts); err != nil {
			log.Printf("API: Consumer error: %v", err)
		}
		log.Printf("API: Consumer goroutine exited")
	}()

	log.Printf("API: Main loop starting to read from posts channel")

	// Broadcast posts to all WebSocket clients
	for post := range posts {
		messageCount++
		now := time.Now()
		if now.Sub(lastLogTime) >= time.Second*10 {
			log.Printf("API: Status update - Received %d messages in last 10 seconds", messageCount)
			messageCount = 0
			lastLogTime = now
		}

		log.Printf("API: ✨ NEW MESSAGE ✨ - Received post from consumer - ID: %s, Title: %s, Subreddit: %s", post.ID, post.Title, post.Subreddit)

		func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			// Add to recent posts buffer
			s.recentPosts = append(s.recentPosts, post)
			if len(s.recentPosts) > 100 {
				s.recentPosts = s.recentPosts[1:] // Remove oldest post
			}
			log.Printf("API: Added post to recent posts buffer (size: %d) - ID: %s", len(s.recentPosts), post.ID)

			// Skip broadcast if no clients
			if len(s.clients) == 0 {
				log.Printf("API: No clients connected, post cached for future clients - ID: %s", post.ID)
				return
			}

			data, err := json.Marshal(post)
			if err != nil {
				log.Printf("API: Error marshaling post: %v", err)
				return
			}

			log.Printf("API: Broadcasting post to %d clients - ID: %s", len(s.clients), post.ID)

			// Create a list of clients to remove
			var clientsToRemove []*websocket.Conn

			// Send to all clients
			for client := range s.clients {
				if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("API: Error sending to client %s: %v", client.RemoteAddr(), err)
					clientsToRemove = append(clientsToRemove, client)
					continue
				}
				log.Printf("API: Successfully sent post to client %s - ID: %s", client.RemoteAddr(), post.ID)
			}

			// Remove failed clients
			for _, client := range clientsToRemove {
				delete(s.clients, client)
				client.Close()
				log.Printf("API: Removed failed client %s", client.RemoteAddr())
			}
		}()
	}
	log.Printf("API: Posts channel closed, consumer loop exiting")
}
