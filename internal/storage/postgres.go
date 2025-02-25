package storage

import (
	"context"
	"database/sql"
	"fmt"
	"goreddit/internal/config"
	"goreddit/internal/reddit"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(cfg *config.Config) (*PostgresStore, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) SavePost(ctx context.Context, post reddit.Post) error {
	/*query := `
		INSERT INTO reddit_posts (
			id, title, body, subreddit, score, url, created_at, sentiment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			score = EXCLUDED.score,
			sentiment = EXCLUDED.sentiment
	`

	_, err := s.db.ExecContext(ctx, query,
		post.ID,
		post.Title,
		post.Body,
		post.Subreddit,
		post.Score,
		post.URL,
		post.CreatedAt,
		post.Sentiment,
	)

	if err != nil {
		return fmt.Errorf("failed to save post: %w", err)
	}
	*/
	return nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
