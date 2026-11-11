package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"termo-sync/internal/domain"
	"time"
)

// Transcribe requires a separately recorded user authorization. POST requests are never retried.
func (c *Client) Transcribe(ctx context.Context, key, path string, authorized bool) (domain.Transcript, error) {
	var t domain.Transcript
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
	if err = c.request(ctx, "POST", "https://api.assemblyai.com/v2/upload", key, f, &upload); err != nil {
		return t, err
	}
	if upload.URL == "" {
		return t, errors.New("upload sem URL")
	}
	body, _ := json.Marshal(map[string]any{"audio_url": upload.URL, "speech_models": []string{"universal-3-5-pro"}, "language_code": "pt", "speaker_labels": true, "disfluencies": true})
	if err = c.request(ctx, "POST", "https://api.assemblyai.com/v2/transcript", key, bytes.NewReader(body), &t); err != nil {
		return t, err
	}
	if !remoteID.MatchString(t.ID) {
		return t, errors.New("transcrição sem identificador")
	}
	id := t.ID
	for {
		switch t.Status {
		case "completed":
			if t.Text == "" {
				return t, errors.New("transcrição vazia")
			}
			return t, nil
		case "error":
			return t, errors.New("transcrição falhou")
		case "queued", "processing":
		default:
			return t, errors.New("estado remoto desconhecido")
		}
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return t, ctx.Err()
		case <-timer.C:
		}
		var next domain.Transcript
		if err = c.request(ctx, "GET", "https://api.assemblyai.com/v2/transcript/"+id, key, nil, &next); err != nil {
			return t, err
		}
		if next.ID != id {
			return t, errors.New("identificador remoto divergente")
		}
		t = next
	}
}

var remoteID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// DeleteTranscript is only called by an explicit user action, never by local cleanup.
func (c *Client) DeleteTranscript(ctx context.Context, key, id string) error {
	if !remoteID.MatchString(id) {
		return errors.New("identificador remoto inválido")
	}
	return c.request(ctx, "DELETE", "https://api.assemblyai.com/v2/transcript/"+id, key, nil, nil)
}
