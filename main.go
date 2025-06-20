package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rosszurowski/small-seasons-bot/bsky"
	"github.com/rosszurowski/small-seasons-bot/mastodon"
	"golang.org/x/sync/errgroup"
)

//go:embed sekki.json
var sekkiJSON string

var dev = flag.Bool("dev", false, "run in dev mode")

var (
	ErrAlreadyPosted = errors.New("already posted")
	ErrNoSeason      = errors.New("no season to post")
)

type rawSeason struct {
	ID          string
	Title       string
	Description string
	StartDate   string
	Emoji       string
}

type Season struct {
	ID      string
	Date    time.Time // date this year to post the post at
	Content string    // raw post text
}

func main() {
	flag.Parse()

	seasons, err := loadSeasons()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	var wg errgroup.Group
	wg.Go(func() error {
		baseURL := os.Getenv("MASTODON_BASE_URL")
		if baseURL == "" {
			log.Println("No MASTODON_BASE_URL, skipping…")
			return nil
		}
		accessToken := os.Getenv("MASTODON_ACCESS_TOKEN")
		if accessToken == "" {
			log.Println("No MASTODON_ACCESS_TOKEN, skipping…")
			return nil
		}

		client, err := mastodon.NewClient(mastodon.Config{
			BaseURL:     baseURL,
			AccessToken: accessToken,
		})
		if err != nil {
			return fmt.Errorf("creating mastodon client: %w", err)
		}
		if err := postToMastodon(context.Background(), client, seasons, now); err != nil {
			return fmt.Errorf("posting to mastodon: %w", err)
		}
		return nil
	})
	wg.Go(func() error {
		handle := os.Getenv("BSKY_HANDLE")
		if handle == "" {
			log.Println("No BSKY_HANDLE, skipping…")
			return nil
		}
		apiKey := os.Getenv("BSKY_API_KEY")
		if apiKey == "" {
			log.Println("No BSKY_API_KEY, skipping…")
			return nil
		}
		ctx := context.Background()
		client, err := bsky.NewClient(ctx, handle, apiKey)
		if err != nil {
			return fmt.Errorf("creating bsky client: %w", err)
		}
		if err := postToBsky(context.Background(), client, seasons, now); err != nil {
			return fmt.Errorf("posting to bsky: %w", err)
		}
		return nil
	})
	if err := wg.Wait(); err != nil {
		log.Fatal(err)
	}
}

// loadSeasons gets a list of seasons, with dates formatted for the current year.
func loadSeasons() ([]Season, error) {
	var rs []rawSeason
	err := json.Unmarshal([]byte(sekkiJSON), &rs)
	if err != nil {
		return nil, fmt.Errorf("error loading sekki: %w", err)
	}

	var seasons []Season
	for _, s := range rs {
		year := time.Now().Year()
		hour := "16:02:00" // just so it's not at the beginning of the day
		dateThisYear, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%s %s", year, s.StartDate, hour))
		if err != nil {
			return nil, fmt.Errorf("error parsing date: %w", err)
		}
		season := Season{
			ID:      s.ID,
			Date:    dateThisYear,
			Content: fmt.Sprintf("%s. %s %s", s.Title, s.Description, s.Emoji),
		}
		seasons = append(seasons, season)
	}
	return seasons, nil
}

// getPostableSeason returns the season that should be posted, or an error if
// there's nothing to post.
func getPostableSeason(seasons []Season, now time.Time, latestTimestamps []time.Time) (Season, error) {
	oneDayAgo := now.Add(time.Hour * -24)
	oneDayFromNow := now.Add(time.Hour * 24)
	for _, s := range seasons {
		if s.Date.Before(oneDayAgo) {
			// We're running this cron job more frequently than every 24 hours,
			// so ignore dates before then.
			continue
		}
		if s.Date.After(oneDayFromNow) {
			// Ignore dates in the future. Our seasons are weeks apart, so this
			// rough check should never be an issue.
			continue
		}
		for _, t := range latestTimestamps {
			y1, m1, d1 := t.Date()
			y2, m2, d2 := s.Date.Date()
			y3, m3, d3 := now.Date()
			postedOnDate := y1 == y2 && m1 == m2 && d1 == d2
			postedToday := y1 == y3 && m1 == m3 && d1 == d3
			if postedOnDate || postedToday {
				// If we've already posted on the date, don't post again.
				return Season{}, ErrAlreadyPosted
			}
		}
		if s.Date.Sub(now) < time.Hour {
			// If we're within an hour of the expected time, post it!
			return s, nil
		}
	}
	return Season{}, ErrNoSeason
}

func postToBsky(ctx context.Context, client *bsky.Client, seasons []Season, now time.Time) error {
	posts, err := client.GetPosts(ctx)
	if err != nil {
		return fmt.Errorf("getting posts: %w", err)
	}
	var timestamps []time.Time
	for _, post := range posts {
		log.Println("found posts", post.CID, post.AuthorDid, post.AuthorHandle, post.Created)
		timestamps = append(timestamps, post.Created)
	}
	season, err := getPostableSeason(seasons, now, timestamps)
	if err != nil {
		if errors.Is(err, ErrAlreadyPosted) {
			log.Println("bsky: already posted today")
			return nil
		} else if errors.Is(err, ErrNoSeason) {
			log.Println("bsky: no season to post")
			return nil
		}
		return fmt.Errorf("getting postable season: %w", err)
	}
	if *dev {
		log.Printf("bsky: would post %s (skipping in dev mode)", season.ID)
		return nil
	}
	log.Printf("bsky: posting %s", season.ID)
	post, err := bsky.NewPostBuilder().
		AddText(season.Content).
		Build()
	if err != nil {
		return fmt.Errorf("building post: %w", err)
	}
	_, err = client.PostToFeed(ctx, post)
	if err != nil {
		return fmt.Errorf("posting to bsky: %w", err)
	}
	return nil
}

func postToMastodon(ctx context.Context, client *mastodon.Client, seasons []Season, now time.Time) error {
	latest, err := client.UserTimeline(ctx)
	if err != nil {
		return fmt.Errorf("getting latest toots: %w", err)
	}
	var timestamps []time.Time
	for _, toot := range latest {
		timestamps = append(timestamps, toot.Created)
	}
	season, err := getPostableSeason(seasons, now, timestamps)
	if err != nil {
		if errors.Is(err, ErrAlreadyPosted) {
			log.Println("mastodon: already posted today")
			return nil
		} else if errors.Is(err, ErrNoSeason) {
			log.Println("mastodon: no season to post")
			return nil
		}
		return fmt.Errorf("getting postable season: %w", err)
	}
	if *dev {
		log.Printf("mastodon: would post %s (skipping in dev mode)", season.ID)
		return nil
	}
	log.Printf("mastodon: posting %s", season.ID)
	status, err := client.PostStatus(ctx, mastodon.PostStatusParams{
		Status: season.Content,
	})
	if err != nil {
		return fmt.Errorf("posting to mastodon: %w", err)
	}
	log.Printf("mastodon: posted! %s", status.URL)

	return nil
}
