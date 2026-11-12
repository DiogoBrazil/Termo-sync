package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"termo-sync/internal/domain"
	"time"
)

// Progress receives the current stage and a fraction in [0,1], or -1 when no real measure exists.
type Progress func(stage string, fraction float64)

type countingReader struct {
	r      io.Reader
	done   int64
	total  int64
	report Progress
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.done += int64(n)
	if c.total > 0 {
		c.report("uploading", float64(c.done)/float64(c.total))
	}
	return n, err
}

// Transcribe requires a separately recorded user authorization. POST requests are never retried.
// Only the idempotent status query is repeated after transient failures.
func (c *Client) Transcribe(ctx context.Context, key, path string, authorized bool, report Progress) (domain.Transcript, error) {
	var t domain.Transcript
	if report == nil {
		report = func(string, float64) {}
	}
	if !authorized {
		return t, errors.New("envio não autorizado")
	}
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()
	var upload struct {
		URL string `json:"upload_url"`
	}
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return t, errors.New("áudio preparado vazio ou ilegível")
	}
	report("uploading", 0)
	body := &countingReader{r: f, total: stat.Size(), report: report}
	if err = c.send(ctx, "POST", "https://api.assemblyai.com/v2/upload", key, body, stat.Size(), &upload); err != nil {
		return t, err
	}
	report("transcribing", -1)
	if upload.URL == "" {
		return t, errors.New("upload sem URL")
	}
	payload, _ := json.Marshal(map[string]any{"audio_url": upload.URL, "speech_models": []string{"universal-3-5-pro"}, "language_code": "pt", "speaker_labels": true, "disfluencies": true})
	if err = c.request(ctx, "POST", "https://api.assemblyai.com/v2/transcript", key, bytes.NewReader(payload), &t); err != nil {
		return t, err
	}
	if !remoteID.MatchString(t.ID) {
		return t, errors.New("transcrição sem identificador")
	}
	id := t.ID
	failures := 0
	for {
		switch t.Status {
		case "completed":
			if t.Text == "" {
				return t, errors.New("transcrição vazia")
			}
			return t, nil
		case "error":
			if reason := clip(t.Error); reason != "" {
				return t, errors.New("a transcrição não foi concluída: " + reason)
			}
			return t, errors.New("a transcrição não foi concluída")
		case "queued", "processing":
		default:
			return t, errors.New("estado remoto desconhecido")
		}
		wait := c.pollInterval()
		var next domain.Transcript
		for {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return t, ctx.Err()
			case <-timer.C:
			}
			err = c.request(ctx, "GET", "https://api.assemblyai.com/v2/transcript/"+id, key, nil, &next)
			if err == nil {
				failures = 0
				break
			}
			// The remote job keeps running (and being billed): tolerate brief outages before giving up.
			var status *StatusError
			transient := errors.Is(err, errConnection) || (errors.As(err, &status) && status.Transient())
			failures++
			if ctx.Err() != nil || !transient || failures > 20 {
				return t, err
			}
			wait = min(c.pollInterval()*time.Duration(1<<min(failures, 4)), time.Minute)
			if status != nil && status.RetryAfter > wait {
				wait = status.RetryAfter
			}
		}
		if next.ID != id {
			return t, errors.New("identificador remoto divergente")
		}
		t = next
	}
}

func (c *Client) pollInterval() time.Duration {
	if c.Poll > 0 {
		return c.Poll
	}
	return 3 * time.Second
}

func clip(text string) string {
	if r := []rune(text); len(r) > 300 {
		return string(r[:300]) + "…"
	}
	return text
}

var remoteID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// DeleteTranscript is only called by an explicit user action, never by local cleanup.
func (c *Client) DeleteTranscript(ctx context.Context, key, id string) error {
	if !remoteID.MatchString(id) {
		return errors.New("identificador remoto inválido")
	}
	return c.request(ctx, "DELETE", "https://api.assemblyai.com/v2/transcript/"+id, key, nil, nil)
}
