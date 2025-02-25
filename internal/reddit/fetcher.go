package reddit

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type PostChannel chan Post

// List of news and politics related subreddits
var TargetSubreddits = []string{
	"news",
	"worldnews",
	"politics",
	"geopolitics",
	"neutralnews",
	"worldpolitics",
	"internationalnews",
	"moderatepolitics",
	"politicaldiscussion",
	"anime_titties", // Despite the name, this is actually a serious world news subreddit
}

func (c *Client) StreamPosts(ctx context.Context, posts PostChannel) {
	log.Printf("Starting to poll the following subreddits: %s", strings.Join(TargetSubreddits, ", "))
	seenPosts := make(map[string]bool)

	// Create a map for O(1) lookup of valid subreddits
	validSubreddits := make(map[string]bool)
	for _, sub := range TargetSubreddits {
		validSubreddits[strings.ToLower(sub)] = true
	}

	// Stream new posts
	log.Printf("Starting to stream new posts from target subreddits...")
	for {
		select {
		case <-ctx.Done():
			close(posts)
			return
		default:
			opts := reddit.ListOptions{
				Limit: 100, // Maximum allowed by Reddit API
			}

			// Join subreddits with + for multi-subreddit query
			subredditQuery := strings.Join(TargetSubreddits, "+")
			submissions, _, err := c.client.Subreddit.NewPosts(ctx, subredditQuery, &opts)
			if err != nil {
				log.Printf("Error fetching posts: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, submission := range submissions {
				// Skip if we've seen this post or if it's not from our target subreddits
				if seenPosts[submission.ID] || !validSubreddits[strings.ToLower(submission.SubredditName)] {
					if !validSubreddits[strings.ToLower(submission.SubredditName)] {
						log.Printf("Skipping post from non-target subreddit: r/%s", submission.SubredditName)
					}
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
					log.Printf("Sent new post from r/%s: %s", post.Subreddit, post.Title)
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
