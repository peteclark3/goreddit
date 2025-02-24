package reddit

import (
	"fmt"
	"goreddit/internal/config"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Client struct {
	client *reddit.Client
	cfg    *config.Config
}

// NewClient creates a new Reddit client
func NewClient(cfg *config.Config) (*Client, error) {
	credentials := reddit.Credentials{
		ID:       cfg.Reddit.ClientID,
		Secret:   cfg.Reddit.ClientSecret,
		Username: cfg.Reddit.Username,
		Password: cfg.Reddit.Password,
	}

	client, err := reddit.NewClient(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reddit client: %w", err)
	}

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// Post represents a Reddit post with the fields we care about
type Post struct {
	ID        string
	Title     string
	Body      string
	Subreddit string
	Score     int32
	URL       string
	CreatedAt float64
	Sentiment float64  // -1.0 to 1.0 sentiment score
	Topics    []string // Add this field
}

// TopicCount represents a topic and its frequency
type TopicCount struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}
