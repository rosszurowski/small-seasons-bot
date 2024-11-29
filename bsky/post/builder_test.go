package post

import (
	"testing"

	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/watzon/lining/models"
)

func TestBuilder(t *testing.T) {
	t.Run("creates empty post", func(t *testing.T) {
		post, err := NewBuilder().Build()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "" {
			t.Errorf("expected empty text, got %v", post.Text)
		}
		if len(post.Facets) != 0 {
			t.Errorf("expected empty facets, got %v", post.Facets)
		}
	})

	t.Run("creates post with text", func(t *testing.T) {
		post, err := NewBuilder().AddText("Hello world").Build()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello world" {
			t.Errorf("expected text 'Hello world', got %v", post.Text)
		}
		if len(post.Facets) != 0 {
			t.Errorf("expected empty facets, got %v", post.Facets)
		}
	})

	t.Run("handles mentions", func(t *testing.T) {
		post, err := NewBuilder().
			AddText("Hello ").
			AddMention("alice", "did:plc:alice").
			AddText("!").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello @alice!" {
			t.Errorf("expected text 'Hello @alice!', got %v", post.Text)
		}
		if len(post.Facets) != 1 {
			t.Errorf("expected 1 facet, got %v", len(post.Facets))
		}

		facet := post.Facets[0]
		if facet.Index.ByteStart != int64(6) {
			t.Errorf("expected ByteStart 6, got %v", facet.Index.ByteStart)
		}
		if facet.Index.ByteEnd != int64(12) {
			t.Errorf("expected ByteEnd 12, got %v", facet.Index.ByteEnd)
		}
		if facet.Features[0].RichtextFacet_Mention == nil {
			t.Errorf("expected mention feature, got nil")
		}
		if facet.Features[0].RichtextFacet_Mention.Did != "did:plc:alice" {
			t.Errorf("expected Did 'did:plc:alice', got %v", facet.Features[0].RichtextFacet_Mention.Did)
		}
	})

	t.Run("handles single hashtags", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			wantText string
			wantTag  string
		}{
			{
				name:     "with hash prefix",
				input:    "#golang",
				wantText: "#golang",
				wantTag:  "golang",
			},
			{
				name:     "without hash prefix",
				input:    "golang",
				wantText: "#golang",
				wantTag:  "golang",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				post, err := NewBuilder().AddTag(tt.input).Build()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if post.Text != tt.wantText {
					t.Errorf("expected text %v, got %v", tt.wantText, post.Text)
				}
				if len(post.Facets) != 1 {
					t.Errorf("expected 1 facet, got %v", len(post.Facets))
				}

				facet := post.Facets[0]
				if facet.Index.ByteStart != int64(0) {
					t.Errorf("expected ByteStart 0, got %v", facet.Index.ByteStart)
				}
				if facet.Index.ByteEnd != int64(len(tt.wantText)) {
					t.Errorf("expected ByteEnd %v, got %v", len(tt.wantText), facet.Index.ByteEnd)
				}
				if facet.Features[0].RichtextFacet_Tag == nil {
					t.Errorf("expected tag feature, got nil")
				}
				if facet.Features[0].RichtextFacet_Tag.Tag != tt.wantTag {
					t.Errorf("expected Tag %v, got %v", tt.wantTag, facet.Features[0].RichtextFacet_Tag.Tag)
				}
			})
		}
	})

	t.Run("handles double hashtags", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			wantText string
			wantTag  string
		}{
			{
				name:     "with double hash prefix",
				input:    "##meta",
				wantText: "##meta",
				wantTag:  "meta",
			},
			{
				name:     "with single hash prefix",
				input:    "#meta",
				wantText: "#meta",
				wantTag:  "meta",
			},
			{
				name:     "without hash prefix",
				input:    "meta",
				wantText: "#meta",
				wantTag:  "meta",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				post, err := NewBuilder().AddTag(tt.input).Build()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if post.Text != tt.wantText {
					t.Errorf("expected text %v, got %v", tt.wantText, post.Text)
				}
				if len(post.Facets) != 1 {
					t.Errorf("expected 1 facet, got %v", len(post.Facets))
				}

				facet := post.Facets[0]
				if facet.Index.ByteStart != int64(0) {
					t.Errorf("expected ByteStart 0, got %v", facet.Index.ByteStart)
				}
				if facet.Index.ByteEnd != int64(len(tt.wantText)) {
					t.Errorf("expected ByteEnd %v, got %v", len(tt.wantText), facet.Index.ByteEnd)
				}
				if facet.Features[0].RichtextFacet_Tag == nil {
					t.Errorf("expected tag feature, got nil")
				}
				if facet.Features[0].RichtextFacet_Tag.Tag != tt.wantTag {
					t.Errorf("expected Tag %v, got %v", tt.wantTag, facet.Features[0].RichtextFacet_Tag.Tag)
				}
			})
		}
	})

	t.Run("handles links", func(t *testing.T) {
		t.Run("with custom text", func(t *testing.T) {
			post, err := NewBuilder().
				AddLink("click here", "https://example.com").
				Build()

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if post.Text != "click here" {
				t.Errorf("expected text 'click here', got %v", post.Text)
			}
			if len(post.Facets) != 1 {
				t.Errorf("expected 1 facet, got %v", len(post.Facets))
			}

			facet := post.Facets[0]
			if facet.Index.ByteStart != int64(0) {
				t.Errorf("expected ByteStart 0, got %v", facet.Index.ByteStart)
			}
			if facet.Index.ByteEnd != int64(10) {
				t.Errorf("expected ByteEnd 10, got %v", facet.Index.ByteEnd)
			}
			if facet.Features[0].RichtextFacet_Link == nil {
				t.Errorf("expected link feature, got nil")
			}
			if facet.Features[0].RichtextFacet_Link.Uri != "https://example.com" {
				t.Errorf("expected Uri 'https://example.com', got %v", facet.Features[0].RichtextFacet_Link.Uri)
			}
		})

		t.Run("with URL as text", func(t *testing.T) {
			url := "https://example.com"
			post, err := NewBuilder().
				AddURLLink(url).
				Build()

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if post.Text != url {
				t.Errorf("expected text %v, got %v", url, post.Text)
			}
			if len(post.Facets) != 1 {
				t.Errorf("expected 1 facet, got %v", len(post.Facets))
			}

			facet := post.Facets[0]
			if facet.Index.ByteStart != int64(0) {
				t.Errorf("expected ByteStart 0, got %v", facet.Index.ByteStart)
			}
			if facet.Index.ByteEnd != int64(len(url)) {
				t.Errorf("expected ByteEnd %v, got %v", len(url), facet.Index.ByteEnd)
			}
			if facet.Features[0].RichtextFacet_Link == nil {
				t.Errorf("expected link feature, got nil")
			}
			if facet.Features[0].RichtextFacet_Link.Uri != url {
				t.Errorf("expected Uri %v, got %v", url, facet.Features[0].RichtextFacet_Link.Uri)
			}
		})
	})

	t.Run("handles multiple facets", func(t *testing.T) {
		post, err := NewBuilder().
			AddText("Hello ").
			AddMention("alice", "did:plc:alice").
			AddText("! Check out ").
			AddLink("this link", "https://example.com").
			AddText(" about ").
			AddTag("#golang").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expectedText := "Hello @alice! Check out this link about #golang"
		if post.Text != expectedText {
			t.Errorf("expected text %v, got %v", expectedText, post.Text)
		}
		if len(post.Facets) != 3 {
			t.Errorf("expected 3 facets, got %v", len(post.Facets))
		}

		// Check mention
		if post.Facets[0].Features[0].RichtextFacet_Mention == nil {
			t.Errorf("expected mention feature, got nil")
		}
		if post.Facets[0].Features[0].RichtextFacet_Mention.Did != "did:plc:alice" {
			t.Errorf("expected Did 'did:plc:alice', got %v", post.Facets[0].Features[0].RichtextFacet_Mention.Did)
		}

		// Check link
		if post.Facets[1].Features[0].RichtextFacet_Link == nil {
			t.Errorf("expected link feature, got nil")
		}
		if post.Facets[1].Features[0].RichtextFacet_Link.Uri != "https://example.com" {
			t.Errorf("expected Uri 'https://example.com', got %v", post.Facets[1].Features[0].RichtextFacet_Link.Uri)
		}

		// Check hashtag
		if post.Facets[2].Features[0].RichtextFacet_Tag == nil {
			t.Errorf("expected tag feature, got nil")
		}
		if post.Facets[2].Features[0].RichtextFacet_Tag.Tag != "golang" {
			t.Errorf("expected Tag 'golang', got %v", post.Facets[2].Features[0].RichtextFacet_Tag.Tag)
		}
	})

	t.Run("handles spaces and newlines", func(t *testing.T) {
		post, err := NewBuilder().
			AddText("Line 1").
			AddNewLine().
			AddText("Line 2").
			AddSpace().
			AddText("continued").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expectedText := "Line 1\nLine 2 continued"
		if post.Text != expectedText {
			t.Errorf("expected text %v, got %v", expectedText, post.Text)
		}
		if len(post.Facets) != 0 {
			t.Errorf("expected empty facets, got %v", post.Facets)
		}
	})

	t.Run("validation", func(t *testing.T) {
		t.Run("post length", func(t *testing.T) {
			// Create a string that exceeds maxPostLength
			longText := make([]byte, maxPostLength+1)
			for i := range longText {
				longText[i] = 'a'
			}

			post, err := NewBuilder().
				AddText(string(longText)).
				Build()

			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if err != ErrPostTooLong {
				t.Errorf("expected ErrPostTooLong, got %v", err)
			}
			if post.Text != "" {
				t.Errorf("expected empty text, got %v", post.Text)
			}
		})

		t.Run("invalid mention", func(t *testing.T) {
			tests := []struct {
				name     string
				username string
			}{
				{
					name:     "empty username",
					username: "",
				},
				{
					name:     "username with spaces",
					username: "user name",
				},
				{
					name:     "username with @",
					username: "@username",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					post, err := NewBuilder().
						AddMention(tt.username, "did:plc:test").
						Build()

					if err == nil {
						t.Fatalf("expected error, got none")
					}
					if err != ErrInvalidMention {
						t.Errorf("expected ErrInvalidMention, got %v", err)
					}
					if post.Text != "" {
						t.Errorf("expected empty text, got %v", post.Text)
					}
				})
			}
		})

		t.Run("invalid tag", func(t *testing.T) {
			tests := []struct {
				name string
				tag  string
			}{
				{
					name: "empty tag",
					tag:  "",
				},
				{
					name: "tag with spaces",
					tag:  "tag with spaces",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					post, err := NewBuilder().
						AddTag(tt.tag).
						Build()

					if err == nil {
						t.Fatalf("expected error, got none")
					}
					if err != ErrInvalidTag {
						t.Errorf("expected ErrInvalidTag, got %v", err)
					}
					if post.Text != "" {
						t.Errorf("expected empty text, got %v", post.Text)
					}
				})
			}
		})

		t.Run("invalid URL", func(t *testing.T) {
			tests := []struct {
				name string
				url  string
			}{
				{
					name: "empty URL",
					url:  "",
				},
				{
					name: "invalid scheme",
					url:  "not-a-url",
				},
				{
					name: "missing host",
					url:  "http://",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Run("AddLink", func(t *testing.T) {
						post, err := NewBuilder().
							AddLink("click here", tt.url).
							Build()

						if err == nil {
							t.Fatalf("expected error, got none")
						}
						if err != ErrInvalidURL {
							t.Errorf("expected ErrInvalidURL, got %v", err)
						}
						if post.Text != "" {
							t.Errorf("expected empty text, got %v", post.Text)
						}
					})

					t.Run("AddURLLink", func(t *testing.T) {
						post, err := NewBuilder().
							AddURLLink(tt.url).
							Build()

						if err == nil {
							t.Fatalf("expected error, got none")
						}
						if err != ErrInvalidURL {
							t.Errorf("expected ErrInvalidURL, got %v", err)
						}
						if post.Text != "" {
							t.Errorf("expected empty text, got %v", post.Text)
						}
					})
				})
			}
		})

		t.Run("mismatched images", func(t *testing.T) {
			post, err := NewBuilder().
				WithImages([]lexutil.LexBlob{{}}, []models.Image{}).
				Build()

			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if err != ErrMismatchedImages {
				t.Errorf("expected ErrMismatchedImages, got %v", err)
			}
			if post.Text != "" {
				t.Errorf("expected empty text, got %v", post.Text)
			}
		})

		t.Run("error propagation", func(t *testing.T) {
			// Test that once an error occurs, subsequent operations are skipped
			post, err := NewBuilder().
				AddText("Hello ").
				AddMention("invalid user", "did:plc:test"). // This will fail
				AddText(" and more text").                  // This should be skipped
				Build()

			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if err != ErrInvalidMention {
				t.Errorf("expected ErrInvalidMention, got %v", err)
			}
			if post.Text != "" {
				t.Errorf("expected empty text, got %v", post.Text)
			}
		})
	})
}

func TestBuilderJoinStrategies(t *testing.T) {
	t.Run("JoinAsIs strategy", func(t *testing.T) {
		post, err := NewBuilder().
			AddText("Hello").
			AddText("world").
			AddText("!").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Helloworld!" {
			t.Errorf("expected text 'Helloworld!', got %v", post.Text)
		}
	})

	t.Run("JoinWithSpaces strategy", func(t *testing.T) {
		post, err := NewBuilder(WithJoinStrategy(JoinWithSpaces)).
			AddText("Hello").
			AddText("world").
			AddText("!").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello world !" {
			t.Errorf("expected text 'Hello world !', got %v", post.Text)
		}
	})

	t.Run("JoinWithSpaces with facets", func(t *testing.T) {
		post, err := NewBuilder(WithJoinStrategy(JoinWithSpaces)).
			AddText("Hello").
			AddMention("alice", "did:plc:alice").
			AddText("!").
			AddText("Check out").
			AddLink("this", "https://example.com").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello @alice ! Check out this" {
			t.Errorf("expected text 'Hello @alice ! Check out this', got %v", post.Text)
		}
		if len(post.Facets) != 2 {
			t.Errorf("expected 2 facets, got %v", len(post.Facets))
		}
	})

	t.Run("JoinWithSpaces with empty segments", func(t *testing.T) {
		post, err := NewBuilder(WithJoinStrategy(JoinWithSpaces)).
			AddText("Hello").
			AddText(""). // Empty segment
			AddText("world").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello world" {
			t.Errorf("expected text 'Hello world', got %v", post.Text)
		}
	})
}

func TestBuilderOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		builder := NewBuilder()
		if builder.options.JoinStrategy != JoinAsIs {
			t.Errorf("expected JoinStrategy JoinAsIs, got %v", builder.options.JoinStrategy)
		}
	})

	t.Run("with join strategy option", func(t *testing.T) {
		builder := NewBuilder(WithJoinStrategy(JoinWithSpaces))
		if builder.options.JoinStrategy != JoinWithSpaces {
			t.Errorf("expected JoinStrategy JoinWithSpaces, got %v", builder.options.JoinStrategy)
		}
	})

	t.Run("multiple options (future-proofing)", func(t *testing.T) {
		// This test ensures our options system can handle multiple options
		// when we add more in the future
		builder := NewBuilder(
			WithJoinStrategy(JoinWithSpaces),
			// Add more options here as they're added
		)
		if builder.options.JoinStrategy != JoinWithSpaces {
			t.Errorf("expected JoinStrategy JoinWithSpaces, got %v", builder.options.JoinStrategy)
		}
	})
}

func TestBuilderAutoDetection(t *testing.T) {
	t.Run("auto hashtags", func(t *testing.T) {
		post, err := NewBuilder(WithAutoHashtag(true)).
			AddText("Check out #golang and #programming!").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Check out #golang and #programming!" {
			t.Errorf("expected text 'Check out #golang and #programming!', got %v", post.Text)
		}
		if len(post.Facets) != 2 {
			t.Errorf("expected 2 facets, got %v", len(post.Facets))
		}

		// Verify hashtags
		for _, facet := range post.Facets {
			if facet.Features[0].RichtextFacet_Tag == nil {
				t.Errorf("expected tag feature, got nil")
			}
			tag := facet.Features[0].RichtextFacet_Tag.Tag
			if tag != "golang" && tag != "programming" {
				t.Errorf("unexpected tag %v", tag)
			}
		}
	})

	t.Run("auto mentions", func(t *testing.T) {
		post, err := NewBuilder(WithAutoMention(true)).
			AddText("Hello @alice and @bob!").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hello @alice and @bob!" {
			t.Errorf("expected text 'Hello @alice and @bob!', got %v", post.Text)
		}
		if len(post.Facets) != 2 {
			t.Errorf("expected 2 facets, got %v", len(post.Facets))
		}

		// Verify mentions
		for _, facet := range post.Facets {
			if facet.Features[0].RichtextFacet_Mention == nil {
				t.Errorf("expected mention feature, got nil")
			}
			did := facet.Features[0].RichtextFacet_Mention.Did
			if did != "did:plc:alice" && did != "did:plc:bob" {
				t.Errorf("unexpected DID %v", did)
			}
		}
	})

	t.Run("auto links", func(t *testing.T) {
		post, err := NewBuilder(WithAutoLink(true)).
			AddText("Check https://example.com and https://test.com").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Check https://example.com and https://test.com" {
			t.Errorf("expected text 'Check https://example.com and https://test.com', got %v", post.Text)
		}
		if len(post.Facets) != 2 {
			t.Errorf("expected 2 facets, got %v", len(post.Facets))
		}

		// Verify links
		for _, facet := range post.Facets {
			if facet.Features[0].RichtextFacet_Link == nil {
				t.Errorf("expected link feature, got nil")
			}
			uri := facet.Features[0].RichtextFacet_Link.Uri
			if uri != "https://example.com" && uri != "https://test.com" {
				t.Errorf("unexpected URI %v", uri)
			}
		}
	})

	t.Run("all auto features", func(t *testing.T) {
		post, err := NewBuilder(
			WithAutoHashtag(true),
			WithAutoMention(true),
			WithAutoLink(true),
		).AddText("Hi @alice! Check #golang at https://golang.org #programming").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "Hi @alice! Check #golang at https://golang.org #programming" {
			t.Errorf("expected text 'Hi @alice! Check #golang at https://golang.org #programming', got %v", post.Text)
		}
		if len(post.Facets) != 4 {
			t.Errorf("expected 4 facets, got %v", len(post.Facets))
		}

		var hashtags, mentions, links int
		for _, facet := range post.Facets {
			if facet.Features[0].RichtextFacet_Tag != nil {
				hashtags++
			}
			if facet.Features[0].RichtextFacet_Mention != nil {
				mentions++
			}
			if facet.Features[0].RichtextFacet_Link != nil {
				links++
			}
		}

		if hashtags != 2 {
			t.Errorf("expected 2 hashtags, got %v", hashtags)
		}
		if mentions != 1 {
			t.Errorf("expected 1 mention, got %v", mentions)
		}
		if links != 1 {
			t.Errorf("expected 1 link, got %v", links)
		}
	})

	t.Run("invalid auto-detected items", func(t *testing.T) {
		post, err := NewBuilder(
			WithAutoHashtag(true),
			WithAutoMention(true),
			WithAutoLink(true),
		).AddText("@invalid user #invalid tag https://").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "@invalid user #invalid tag https://" {
			t.Errorf("expected text '@invalid user #invalid tag https://', got %v", post.Text)
		}
		if len(post.Facets) != 0 {
			t.Errorf("expected empty facets, got %v", post.Facets)
		}
	})
}

func TestBuilderMaxLength(t *testing.T) {
	t.Run("custom max length", func(t *testing.T) {
		builder := NewBuilder(WithMaxLength(10))
		post, err := builder.AddText("12345").Build()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if post.Text != "12345" {
			t.Errorf("expected text '12345', got %v", post.Text)
		}

		_, err = builder.AddText("123456").Build()
		if err == nil {
			t.Fatalf("expected error, got none")
		}
		if err != ErrPostTooLong {
			t.Errorf("expected ErrPostTooLong, got %v", err)
		}
	})

	t.Run("invalid max length", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got none")
			}
		}()
		NewBuilder(WithMaxLength(0))

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got none")
			}
		}()
		NewBuilder(WithMaxLength(-1))

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got none")
			}
		}()
		NewBuilder(WithMaxLength(maxPostLength + 1))
	})
}
