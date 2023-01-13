// Package mastodon contains utilities for working with a Mastodon instance API.
package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	http        *http.Client
	baseURL     string
	accessToken string
}

type Config struct {
	Client      *http.Client
	BaseURL     string // BaseURL is the base URL of the Mastodon instance. For example, "https://mastodon.social".
	AccessToken string // AccessToken is the access token for the Mastodon account.
}

// NewClient returns a new Mastodon client
func NewClient(cfg Config) (*Client, error) {
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("AccessToken is required")
	}
	return &Client{
		http:        cfg.Client,
		baseURL:     cfg.BaseURL,
		accessToken: cfg.AccessToken,
	}, nil
}

type Status struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Visibility string    `json:"visibility"`
	Content    string    `json:"content"`
	Created    time.Time `json:"created_at"`
}

// UserTimeline returns the 5 most recent statuses posted by the authenticated user.
func (c *Client) UserTimeline(ctx context.Context) ([]Status, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/timelines/home?limit=5", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	var statuses []Status
	if err := json.NewDecoder(res.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return statuses, nil
}

// PostStatusParams are the parameters for posting a new status.
type PostStatusParams struct {
	Status string `json:"status"`
}

// PostStatus posts a new status to the authenticated user's account.
func (c *Client) PostStatus(ctx context.Context, params PostStatusParams) (*Status, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshalling params: %w", err)
	}
	body := bytes.NewReader(b)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/statuses", body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized")
		case http.StatusUnprocessableEntity:
			return nil, fmt.Errorf("unprocessable entity")
		}
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &status, nil
}
