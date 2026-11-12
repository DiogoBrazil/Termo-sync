package application

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"termo-sync/internal/media"
	"termo-sync/internal/providers"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWorkflowWithSyntheticMediaAndMockProviders(t *testing.T) {
	ffmpeg, ffprobe := os.Getenv("DEPOIMENTO_TEST_FFMPEG"), os.Getenv("DEPOIMENTO_TEST_FFPROBE")
	if ffmpeg == "" || ffprobe == "" {
		t.Skip("configure ferramentas para teste integrado de mídia")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	// 0.1 seconds of silence, WAV PCM mono, no personal content.
	var wav bytes.Buffer
	wav.WriteString("RIFF")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(36+1600))
	wav.WriteString("WAVEfmt ")
	for _, n := range []any{uint32(16), uint16(1), uint16(1), uint32(8000), uint32(16000), uint16(2), uint16(16)} {
		_ = binary.Write(&wav, binary.LittleEndian, n)
	}
	wav.WriteString("data")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(1600))
	wav.Write(make([]byte, 1600))
	path := filepath.Join(t.TempDir(), "ficticio.wav")
	if err := os.WriteFile(path, wav.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	client := providers.New()
	posts := 0
	client.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
		data := `{"upload_url":"https://cdn.assemblyai.com/upload/test"}`
		if r.Method == "POST" {
			posts++
		}
		switch r.URL.Path {
		case "/v2/upload":
		case "/v2/transcript":
			data = `{"id":"abc","status":"completed","text":"Não vi 12 pessoas.","utterances":[{"speaker":"A","text":"Não vi 12 pessoas.","start":0,"end":100},{"speaker":"B","text":"Ouvi dizer.","start":100,"end":200}]}`
		case "/chat/completions":
			var body struct {
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content := body.Messages[1].Content
			response := map[string]any{"model": "deepseek-flash", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"role": "assistant", "content": content}}}, "usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 10, "total_tokens": 20}}
			b, _ := json.Marshal(response)
			data = string(b)
		case "/v2/transcript/abc":
			data = `{"id":"abc","status":"deleted"}`
		default:
			t.Error("unexpected provider URL")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(data)), Request: r}, nil
	})
	s := NewService(memoryStore{"assemblyai": "key", "deepseek": "key"}, client, media.Tools{FFmpeg: ffmpeg, FFprobe: ffprobe})
	defer s.Shutdown()
	if err := s.Open(path); err != nil {
		t.Fatal(err)
	}
	id := s.View().Document.ID
	if err := s.Start(id); err != nil {
		t.Fatal(err)
	}
	await := func(state State) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			v := s.View()
			if !v.Busy {
				if v.Document.State != state {
					t.Fatalf("state %s, error %s", v.Document.State, v.Error)
				}
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("workflow timeout")
	}
	await(Ready)
	if err := s.Edit(id, "Edição humana."); err != nil {
		t.Fatal(err)
	}
	text, err := s.Text()
	if err != nil || text != "Edição humana." {
		t.Fatal("human edit lost")
	}
	if s.View().Document.Proposal != "Não vi 12 pessoas.\n\nOuvi dizer." {
		t.Fatalf("proposal must join every segment without speaker labels: %q", s.View().Document.Proposal)
	}
	if original, _ := s.OriginalText(); original != "[00:00] Não vi 12 pessoas.\n\n[00:00] Ouvi dizer." {
		t.Fatalf("original text: %q", original)
	}
	if s.View().Document.Original.Text != "Não vi 12 pessoas." {
		t.Fatal("original changed")
	}
	if posts != 3 {
		t.Fatalf("unexpected duplicate POST: %d", posts)
	}
	if err := s.DeleteRemote(id, false); err == nil {
		t.Fatal("unconfirmed remote deletion")
	}
	if err := s.DeleteRemote(id, true); err != nil {
		t.Fatal(err)
	}
	if !s.View().RemoteDeleted {
		t.Fatal("deletion not reported")
	}
}
