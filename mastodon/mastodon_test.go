package mastodon

import (
	"context"
	"os"
	"testing"
)

func TestUserTimeline(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:     os.Getenv("MASTODON_BASE_URL"),
		AccessToken: os.Getenv("MASTODON_ACCESS_TOKEN"),
	})
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := client.UserTimeline(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("statuses: %v", statuses)
}
