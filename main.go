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
	ID    string
	Date  time.Time // date this year to post the tweet at
	Tweet string    // raw tweet text
}

func main() {
	seasons, err := loadSeasons()
	if err != nil {
		log.Fatal(err)
	}
	client, err := getTwitterClient()
	if err != nil {
		log.Fatal(err)
	}

	tweetableSeason, err := getTweetableSeason(client, seasons, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	if tweetableSeason == nil {
		log.Println("nothing to tweet")
		os.Exit(0)
	}

	log.Printf("tweeting %s", tweetableSeason.ID)
	tweet, _, err := client.Statuses.Update(tweetableSeason.Tweet, nil)
	if err != nil {
		log.Fatalf("failed to post tweet: %v", err)
	}

	log.Printf("posted tweet! %s", tweet.IDStr)
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
			ID:    s.ID,
			Date:  dateThisYear,
			Tweet: fmt.Sprintf("%s. %s %s", s.Title, s.Description, s.Emoji),
		}
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", s.ID, err)
		}
		seasons = append(seasons, season)
	}
	return seasons, nil
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

func getTweetableSeason(client *twitter.Client, seasons []*Season, now time.Time) (*Season, error) {
	twitterUsername := os.Getenv("TWITTER_USERNAME")
	if twitterUsername == "" {
		return nil, errors.New("TWITTER_USERNAME is missing")
	}

	latestTweets, _, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ScreenName: twitterUsername,
		Count:      2,
	})
	if err != nil {
		return nil, err
	}

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
		alreadyTweeted := false
		for _, tweet := range latestTweets {
			t, err := tweet.CreatedAtTime()
			if err != nil {
				log.Fatalf("failed to parse tweet date: %v", err)
			}
			y1, m1, d1 := t.Date()
			y2, m2, d2 := s.Date.Date()
			fmt.Println(t, s.Date, t.Sub(s.Date))
			if y1 == y2 && m1 == m2 && d1 == d2 {
				// If we've already tweeted today, don't tweet again.
				alreadyTweeted = true
				break
			}
		}
		if alreadyTweeted {
			return nil, nil
		}
		if s.Date.Sub(now) < time.Hour {
			// If we're within an hour of the expected time, tweet it!
			return s, nil
		}
	}
	return nil, nil
}
