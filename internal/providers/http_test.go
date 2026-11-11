package providers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestHTTPValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"valid", `{"id":"abc"}`, 200, false},
		{"invalid", `{`, 200, true}, {"trailing", `{} {}`, 200, true}, {"null", `null`, 200, true},
		{"auth", `secret response`, 401, true}, {"balance", `{}`, 402, true}, {"limit", `{}`, 429, true},
		{"server", `{}`, 503, true}, {"redirect", `{}`, 302, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := New()
			c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "private" {
					t.Fatal("missing auth")
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"https://evil.example"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})
			var out struct {
				ID string `json:"id"`
			}
			err := c.request(context.Background(), "POST", "https://api.assemblyai.com/v2/transcript", "private", strings.NewReader(`{}`), &out)
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected error: %v", err)
			}
			if calls != 1 {
				t.Fatalf("POST repeated %d times", calls)
			}
			if err != nil && (strings.Contains(err.Error(), "secret response") || strings.Contains(err.Error(), "private")) {
				t.Fatal("sensitive error")
			}
		})
	}
}
func TestHTTPDestinationAndCancellation(t *testing.T) {
	c := New()
	calls := 0
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) { calls++; return nil, r.Context().Err() })
	for _, url := range []string{"http://api.assemblyai.com/v2/upload", "https://evil.example", "https://api.assemblyai.com.evil.example", "https://api.assemblyai.com:443/v2/upload", "https://user@api.assemblyai.com/v2/upload"} {
		if err := c.request(context.Background(), "POST", url, "key", nil, nil); err == nil {
			t.Fatal("destination accepted")
		}
	}
	if calls != 0 {
		t.Fatal("unauthorized request sent")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.request(ctx, "GET", "https://api.assemblyai.com/v2/transcript/id", "key", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if _, err := c.Transcribe(context.Background(), "key", "nonexistent", false); err == nil {
		t.Fatal("missing consent")
	}
}
