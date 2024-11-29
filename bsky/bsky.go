package bsky

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/rosszurowski/small-seasons-bot/bsky/post"
)

type Client struct {
	handle string
	apiKey string

	mu     sync.RWMutex
	client *xrpc.Client
}

const blueSkyServer = "https://bsky.social"

// NewClient returns a new bsky.Client
func NewClient(ctx context.Context, handle, apiKey string) (*Client, error) {
	if handle == "" {
		return nil, fmt.Errorf("handle is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	httpc := &http.Client{
		Timeout: time.Second * 30,
	}
	c := &xrpc.Client{
		Client: httpc,
		Host:   blueSkyServer,
	}
	bc := &Client{client: c, handle: handle, apiKey: apiKey}
	if err := bc.authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticating with bsky: %w", err)
	}
	return bc, nil
}

// Authenticate authenticates the client with the bsky server. This
func (c *Client) authenticate(ctx context.Context) error {
	input := &atproto.ServerCreateSession_Input{
		Identifier: c.handle,
		Password:   c.apiKey,
	}
	session, err := atproto.ServerCreateSession(ctx, c.client, input)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	c.mu.Lock()
	c.client.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}
	c.mu.Unlock()
	return nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	if c.client != nil {
		c.mu.Lock()
		c.client.Auth = nil
		c.client = nil
		c.mu.Unlock()
	}
	return nil
}

type BlueskyPost struct {
	CID          string
	AuthorDid    string
	AuthorHandle string
	Created      time.Time
}

func (c *Client) GetPosts(ctx context.Context) ([]*BlueskyPost, error) {
	profile, err := appbsky.ActorGetProfile(ctx, c.client, c.handle)
	if err != nil {
		return nil, fmt.Errorf("getting profile: %w", err)
	}
	resp, err := appbsky.FeedGetAuthorFeed(ctx, c.client, profile.Did, "", "", false, 10)
	if err != nil {
		return nil, fmt.Errorf("getting posts: %w", err)
	}
	posts := make([]*BlueskyPost, 0, len(resp.Feed))
	for _, feed := range resp.Feed {
		p := feed.Post
		t, err := time.Parse(time.RFC3339, p.IndexedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing post timestamp: %w", err)
		}
		posts = append(posts, &BlueskyPost{
			CID:          p.Cid,
			AuthorDid:    p.Author.Did,
			AuthorHandle: p.Author.Handle,
			Created:      t,
		})
	}
	return posts, nil
}

type PostResponse struct {
	CID string
	URI string
}

func (c *Client) PostToFeed(ctx context.Context, post appbsky.FeedPost) (*PostResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	// Create a new post object
	newPost := &appbsky.FeedPost{
		LexiconTypeID: "app.bsky.feed.post",
		Text:          post.Text,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Embed:         post.Embed,
		Facets:        post.Facets,
		Entities:      post.Entities,
		Labels:        post.Labels,
		Langs:         post.Langs,
		Reply:         post.Reply,
		Tags:          post.Tags,
	}

	resp, err := atproto.RepoCreateRecord(ctx, c.client, &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       c.client.Auth.Did,
		Record:     &lexutil.LexiconTypeDecoder{Val: newPost},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}
	return &PostResponse{
		CID: resp.Cid,
		URI: resp.Uri,
	}, nil
}

// NewPostBuilder creates a new post builder with the specified options
func NewPostBuilder(opts ...post.BuilderOption) *post.Builder {
	return post.NewBuilder(opts...)
}
