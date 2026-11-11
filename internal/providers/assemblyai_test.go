package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscribeAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "áudio fictício.mp3")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	c := New()
	calls := 0
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		data := `{"upload_url":"https://cdn.assemblyai.com/upload/test"}`
		switch r.URL.Path {
		case "/v2/upload":
			if r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Fatal("upload content type")
			}
		case "/v2/transcript":
			data = `{"id":"abc-123","status":"completed","text":"Não.","speech_model_used":"universal-3-5-pro","utterances":[{"speaker":"B","start":0,"end":100,"text":"Não."}]}`
		case "/v2/transcript/abc-123":
			if r.Method != "DELETE" {
				t.Fatal("unexpected method")
			}
			data = `{"id":"abc-123","status":"deleted"}`
		default:
			t.Fatal("unexpected endpoint")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(data)), Header: make(http.Header), Request: r}, nil
	})
	transcript, err := c.Transcribe(context.Background(), "key", path, true)
	if err != nil {
		t.Fatal(err)
	}
	if transcript.ID != "abc-123" || len(transcript.Raw) == 0 || transcript.Model == "" {
		t.Fatal("metadata missing")
	}
	if calls != 2 {
		t.Fatal("POST count")
	}
	if err := c.DeleteTranscript(context.Background(), "key", "../upload"); err == nil {
		t.Fatal("path accepted")
	}
	if err := c.DeleteTranscript(context.Background(), "key", transcript.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPollingAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic.mp3")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			c := New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			posts, gets := 0, 0
			c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				data := `{"upload_url":"https://cdn.assemblyai.com/upload/test"}`
				if r.Method == "POST" {
					posts++
				}
				if r.URL.Path == "/v2/transcript" {
					data = `{"id":"job","status":"queued"}`
					if cancelled {
						cancel()
					}
				}
				if r.Method == "GET" {
					gets++
					data = `{"id":"job","status":"completed","text":"Não.","utterances":[{"speaker":"B","text":"Não."}]}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: r}, nil
			})
			result, err := c.Transcribe(ctx, "key", path, true)
			if cancelled {
				if !errors.Is(err, context.Canceled) || result.ID != "job" || gets != 0 {
					t.Fatalf("cancel result %v", err)
				}
			} else {
				if err != nil || result.Status != "completed" || gets != 1 {
					t.Fatalf("poll failed %v", err)
				}
			}
			if posts != 2 {
				t.Fatal("POST repeated")
			}
		})
	}
}
