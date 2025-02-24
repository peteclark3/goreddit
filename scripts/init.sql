CREATE TABLE IF NOT EXISTS reddit_posts (
    id VARCHAR(255) PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT,
    subreddit VARCHAR(255) NOT NULL,
    score INT NOT NULL,
    url TEXT,
    created_at FLOAT NOT NULL,
    sentiment FLOAT,
    stored_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reddit_posts_subreddit ON reddit_posts(subreddit);
CREATE INDEX idx_reddit_posts_created_at ON reddit_posts(created_at);
CREATE INDEX idx_reddit_posts_score ON reddit_posts(score);
CREATE INDEX idx_reddit_posts_sentiment ON reddit_posts(sentiment); 