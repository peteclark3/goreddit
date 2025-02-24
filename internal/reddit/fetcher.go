package reddit

import (
	"context"
	"log"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type PostChannel chan Post

func (c *Client) StreamPosts(ctx context.Context, posts PostChannel) {
	log.Println("Starting to poll r/all/new (rate limited)")
	seenPosts := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			close(posts)
			return
		default:
			opts := reddit.ListOptions{
				Limit: 100, // Maximum allowed by Reddit API
			}

			submissions, _, err := c.client.Subreddit.NewPosts(ctx, "all", &opts)
			if err != nil {
				log.Printf("Error fetching posts: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, submission := range submissions {
				if seenPosts[submission.ID] {
					continue
				}

				post := Post{
					ID:        submission.ID,
					Title:     submission.Title,
					Body:      submission.Body,
					Subreddit: submission.SubredditName,
					Score:     int32(submission.Score),
					URL:       submission.URL,
					CreatedAt: float64(submission.Created.Unix()),
				}

				select {
				case posts <- post:
					seenPosts[submission.ID] = true
				case <-ctx.Done():
					close(posts)
					return
				}
			}

			// Clean up old posts periodically
			if len(seenPosts) > 10000 {
				newSeen := make(map[string]bool)
				for id := range seenPosts {
					if len(newSeen) < 5000 {
						newSeen[id] = true
					}
				}
				seenPosts = newSeen
			}

			time.Sleep(2 * time.Second) // Respect rate limits
		}
	}
}
