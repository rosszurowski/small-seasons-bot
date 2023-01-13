package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rosszurowski/small-seasons-bot/mastodon"
	"golang.org/x/sync/errgroup"
)

//go:embed sekki.json
var sekkiJSON string

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
	seasons, err := loadSeasons()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	var wg errgroup.Group
	wg.Go(func() error {
		return postToTwitter(seasons, now)
	})
	wg.Go(func() error {
		return postToMastodon(seasons, now)
	})
	if err := wg.Wait(); err != nil {
		log.Fatal(err)
	}
}

// loadSeasons gets a list of seasons, with dates formatted for the current year.
func loadSeasons() ([]*Season, error) {
	var rs []rawSeason
	err := json.Unmarshal([]byte(sekkiJSON), &rs)
	if err != nil {
		return nil, fmt.Errorf("error loading sekki: %w", err)
	}

	var seasons []*Season
	for _, s := range rs {
		year := time.Now().Year()
		hour := "16:02:00" // just so it's not at the beginning of the day
		dateThisYear, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%s %s", year, s.StartDate, hour))
		if err != nil {
			return nil, fmt.Errorf("error parsing date: %w", err)
		}
		season := &Season{
			ID:      s.ID,
			Date:    dateThisYear,
			Content: fmt.Sprintf("%s. %s %s", s.Title, s.Description, s.Emoji),
		}
		seasons = append(seasons, season)
	}
	return seasons, nil
}

var (
	ErrAlreadyPosted = errors.New("already posted")
	ErrNoSeason      = errors.New("no season to post")
)

// getPostableSeason returns the season that should be posted, or an error if
// there's nothing to post.
func getPostableSeason(seasons []*Season, now time.Time, latestTimestamps []time.Time) (*Season, error) {
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
			if y1 == y2 && m1 == m2 && d1 == d2 {
				// If we've already posted today, don't post again.
				return nil, ErrAlreadyPosted
			}
		}
		if s.Date.Sub(now) < time.Hour {
			// If we're within an hour of the expected time, post it!
			return s, nil
		}
	}
	return nil, ErrNoSeason
}

func postToMastodon(seasons []*Season, now time.Time) error {
	ctx := context.Background()
	client, err := mastodon.NewClient(mastodon.Config{
		BaseURL:     os.Getenv("MASTODON_BASE_URL"),
		AccessToken: os.Getenv("MASTODON_ACCESS_TOKEN"),
	})
	if err != nil {
		return fmt.Errorf("creating mastodon client: %w", err)
	}
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

func postToTwitter(seasons []*Season, now time.Time) error {
	username := os.Getenv("TWITTER_USERNAME")
	if username == "" {
		log.Fatal("TWITTER_USERNAME is missing")
	}
	client, err := getTwitterClient()
	if err != nil {
		return fmt.Errorf("getting twitter client: %w", err)
	}
	tweets, _, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		// TODO: we might not need to pass in a username here. The API may default
		// to showing the current user's timeline.
		ScreenName: username,
		Count:      2,
	})
	if err != nil {
		return fmt.Errorf("getting latest tweets: %w", err)
	}
	var timestamps []time.Time
	for _, tweet := range tweets {
		t, err := tweet.CreatedAtTime()
		if err != nil {
			return fmt.Errorf("getting tweet timestamp: %w", err)
		}
		timestamps = append(timestamps, t)
	}
	season, err := getPostableSeason(seasons, now, timestamps)
	if err != nil {
		if errors.Is(err, ErrAlreadyPosted) {
			log.Println("twitter: already posted today")
			return nil
		} else if errors.Is(err, ErrNoSeason) {
			log.Println("twitter: no season to post")
			return nil
		}
		return fmt.Errorf("getting postable season: %w", err)
	}
	log.Printf("twitter: posting %s", season.ID)
	tweet, _, err := client.Statuses.Update(season.Content, nil)
	if err != nil {
		return fmt.Errorf("posting tweet: %w", err)
	}
	log.Printf("twitter: posted! https://twitter.com/%s/status/%s", username, tweet.IDStr)
	return nil
}

func getTwitterClient() (*twitter.Client, error) {
	consumerKey := os.Getenv("TWITTER_CONSUMER_KEY")
	if consumerKey == "" {
		return nil, errors.New("TWITTER_CONSUMER_KEY is missing")
	}
	consumerSecret := os.Getenv("TWITTER_CONSUMER_SECRET")
	if consumerSecret == "" {
		return nil, errors.New("TWITTER_CONSUMER_SECRET is missing")
	}
	accessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, errors.New("TWITTER_ACCESS_TOKEN is missing")
	}
	accessSecret := os.Getenv("TWITTER_ACCESS_SECRET")
	if accessSecret == "" {
		return nil, errors.New("TWITTER_ACCESS_SECRET is missing")
	}

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(context.Background(), token)
	client := twitter.NewClient(httpClient)
	return client, nil
}
