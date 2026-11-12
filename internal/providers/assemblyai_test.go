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
	"time"
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
	transcript, err := c.Transcribe(context.Background(), "key", path, true, nil)
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
			result, err := c.Transcribe(ctx, "key", path, true, nil)
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

func TestPollingSurvivesTransientFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic.mp3")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	c := New()
	c.Poll = time.Millisecond
	posts, gets := 0, 0
	var stages []string
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		status, data := 200, `{"upload_url":"https://cdn.assemblyai.com/upload/test"}`
		if r.Method == "POST" {
			posts++
			if r.URL.Path == "/v2/upload" && r.ContentLength != 4 {
				t.Errorf("upload length %d", r.ContentLength)
			}
		}
		if r.URL.Path == "/v2/transcript" {
			data = `{"id":"job","status":"queued"}`
		}
		if r.Method == "GET" {
			gets++
			switch gets {
			case 1:
				return nil, errors.New("connection reset")
			case 2:
				status, data = 503, `{}`
			case 3:
				status, data = 429, `{}`
			default:
				data = `{"id":"job","status":"completed","text":"Não.","utterances":[{"speaker":"B","text":"Não."}]}`
			}
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: r}, nil
	})
	result, err := c.Transcribe(context.Background(), "key", path, true, func(stage string, _ float64) {
		if len(stages) == 0 || stages[len(stages)-1] != stage {
			stages = append(stages, stage)
		}
	})
	if err != nil || result.Status != "completed" || gets != 4 || posts != 2 {
		t.Fatalf("transient polling failure not tolerated: %v gets=%d posts=%d", err, gets, posts)
	}
	if strings.Join(stages, ",") != "uploading,transcribing" {
		t.Fatalf("stages %v", stages)
	}
}

func TestPollingStopsOnPermanentFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic.mp3")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	c := New()
	c.Poll = time.Millisecond
	gets := 0
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		status, data := 200, `{"upload_url":"https://cdn.assemblyai.com/upload/test"}`
		if r.URL.Path == "/v2/transcript" {
			data = `{"id":"job","status":"queued"}`
		}
		if r.Method == "GET" {
			gets++
			status, data = 401, `{"error":"secret body"}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: r}, nil
	})
	result, err := c.Transcribe(context.Background(), "key", path, true, nil)
	var status *StatusError
	if !errors.As(err, &status) || status.Status != 401 || gets != 1 || result.ID != "job" || strings.Contains(err.Error(), "secret body") {
		t.Fatalf("permanent failure mishandled: %v gets=%d", err, gets)
	}
}

func TestRejectedRequestDiagnostic(t *testing.T) {
	c := New()
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Header: http.Header{"Retry-After": []string{"7"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"unsupported   parameter"}}`)), Request: r}, nil
	})
	err := c.request(context.Background(), "POST", "https://api.deepseek.com/chat/completions", "private", strings.NewReader(`{}`), nil)
	var status *StatusError
	if !errors.As(err, &status) || status.Detail != "unsupported parameter" || status.RetryAfter != 7*time.Second || strings.Contains(err.Error(), "private") {
		t.Fatalf("diagnostic lost: %v", err)
	}
}
