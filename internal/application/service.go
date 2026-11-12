package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"termo-sync/internal/credentials"
	"termo-sync/internal/editorial"
	"termo-sync/internal/media"
	"termo-sync/internal/providers"
	"time"
)

type Settings struct {
	AssemblyAIConfigured bool
	DeepSeekConfigured   bool
	Model                string
}
type View struct {
	Document Snapshot
	// Format, Duration (seconds) and Size (bytes) come from the local ffprobe inspection.
	Format   string
	Duration float64
	Size     int64
	// Stage names the running step; Progress is a real fraction in [0,1] or -1 when not measurable.
	Stage         string
	Progress      float64
	Tracks        []media.Track
	Busy          bool
	Error         string
	Changes       []editorial.Change
	Revisions     []providers.Revision
	RemoteID      string
	RemoteDeleted bool
}

// Service is the typed backend surface. It does not return secrets, filesystem paths to temporaries or raw provider errors.
type Service struct {
	mu            sync.Mutex
	session       Session
	store         credentials.Store
	client        *providers.Client
	tools         media.Tools
	info          media.Info
	cancel        context.CancelFunc
	busy          bool
	message       string
	changes       []editorial.Change
	revisions     []providers.Revision
	model         string
	remoteID      string
	remoteDeleted bool
	stage         string
	progress      float64
}

func (s *Service) report(stage string, fraction float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		s.stage, s.progress = stage, fraction
	}
}

// describe turns internal failures into the message shown to the user.
func describe(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "Processamento cancelado. Nada do que já foi obtido nesta sessão foi descartado."
	case errors.Is(err, context.DeadlineExceeded):
		return "Tempo limite de processamento excedido."
	}
	text := err.Error()
	if r := []rune(text); len(r) > 0 {
		text = strings.ToUpper(string(r[:1])) + string(r[1:])
	}
	if !strings.HasSuffix(text, ".") {
		text += "."
	}
	return text
}

func NewService(store credentials.Store, client *providers.Client, tools media.Tools) *Service {
	return &Service{store: store, client: client, tools: tools, model: "deepseek-flash"}
}
func (s *Service) Settings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, _ := s.store.Get("assemblyai")
	d, _ := s.store.Get("deepseek")
	return Settings{AssemblyAIConfigured: a != "", DeepSeekConfigured: d != "", Model: s.model}
}

// SaveKey never exposes a read-key method to the frontend.
func (s *Service) SaveKey(provider, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("aguarde o processamento antes de alterar credenciais")
	}
	key = strings.TrimSpace(key)
	if strings.ContainsAny(key, "\r\n\x00") {
		return errors.New("API Key inválida")
	}
	return s.store.Set(provider, key)
}
func (s *Service) RemoveKey(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("aguarde o processamento antes de remover credenciais")
	}
	return s.store.Delete(provider)
}
func (s *Service) SetModel(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("processamento em andamento")
	}
	if model != "deepseek-flash" && model != "deepseek-v4-pro" {
		return errors.New("modelo inválido")
	}
	s.model = model
	return nil
}
func (s *Service) View() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracks := append([]media.Track(nil), s.info.Streams...)
	changes := append([]editorial.Change(nil), s.changes...)
	// All nested data returned across the bridge is serialized, but copy here as well for Go callers.
	for i := range changes {
		changes[i].Flags = append([]string(nil), changes[i].Flags...)
	}
	duration, _ := strconv.ParseFloat(s.info.Format.Duration, 64)
	size, _ := strconv.ParseInt(s.info.Format.Size, 10, 64)
	return View{Format: s.info.Format.Name, Duration: duration, Size: size, Stage: s.stage, Progress: s.progress, Revisions: append([]providers.Revision(nil), s.revisions...), Document: s.session.Snapshot(), Tracks: tracks, Busy: s.busy, Error: s.message, Changes: changes, RemoteID: s.remoteID, RemoteDeleted: s.remoteDeleted}
}

// Open is called with a native-dialog path, never an arbitrary path from web content.
func (s *Service) Open(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("cancele e aguarde o trabalho atual")
	}
	if s.remoteID != "" && !s.remoteDeleted {
		return errors.New("antes de trocar de arquivo, exclua o registro do arquivo atual ou confirme que deseja mantê-lo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	info, err := s.tools.Inspect(ctx, path)
	if err != nil {
		return err
	}
	tracks := make([]int, len(info.Streams))
	for i, t := range info.Streams {
		tracks[i] = t.Index
	}
	if _, err := s.session.Open(path, tracks); err != nil {
		return err
	}
	s.info = info
	s.message = ""
	s.changes = nil
	s.revisions = nil
	s.remoteID = ""
	s.remoteDeleted = false
	return nil
}
func (s *Service) KeepRemote(id uint64, confirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy || !confirmed || s.session.Snapshot().ID != id {
		return errors.New("confirmação inválida para manter o registro")
	}
	// This only dismisses the session reminder, never claims remote deletion.
	s.remoteID = ""
	return nil
}
func (s *Service) SelectTrack(id uint64, track int) error { return s.session.SelectTrack(id, track) }
func (s *Service) Edit(id uint64, text string) error      { return s.session.Edit(id, text) }

// OriginalText formats the immutable transcript exactly as displayed, for copying.
func (s *Service) OriginalText() (string, error) {
	t := s.session.Snapshot().Original
	if len(t.Utterances) == 0 {
		if t.Text == "" {
			return "", errors.New("transcrição ainda indisponível")
		}
		return t.Text, nil
	}
	var b strings.Builder
	for i, u := range t.Utterances {
		if i > 0 {
			b.WriteString("\n\n")
		}
		seconds := u.Start / 1000
		fmt.Fprintf(&b, "[%02d:%02d] %s", seconds/60, seconds%60, u.Text)
	}
	return b.String(), nil
}
func (s *Service) Text() (string, error) {
	d := s.session.Snapshot()
	if d.HasEdits {
		return d.Edited, nil
	}
	if d.Proposal != "" {
		return d.Proposal, nil
	}
	return "", errors.New("proposta ainda indisponível")
}

// Start runs the whole processing of the current file: local preparation, transcription and review.
// Both keys are checked first so a paid transcription is not left without its review.
func (s *Service) Start(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("processamento em andamento")
	}
	key, err := s.store.Get("assemblyai")
	if err != nil {
		return err
	}
	reviewKey, err := s.store.Get("deepseek")
	if err != nil {
		return err
	}
	if key == "" || reviewKey == "" {
		return errors.New("configure as chaves de API antes de iniciar")
	}
	if err = s.session.Begin(id); err != nil {
		return err
	}
	d := s.session.Snapshot()
	info := s.info
	model := s.model
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Hour)
	s.cancel = cancel
	s.busy = true
	s.message = ""
	s.stage, s.progress = "validating", -1
	go func() {
		defer cancel()
		path, clean, err := s.tools.Prepare(ctx, d.File, info, d.Track, s.report)
		defer clean()
		if err != nil {
			s.session.TranscriptionFailed(id)
			s.finish(err)
			return
		}
		t, err := s.client.Transcribe(ctx, key, path, true, s.report)
		s.mu.Lock()
		if t.ID != "" {
			s.remoteID = t.ID
			s.remoteDeleted = false
		}
		s.mu.Unlock()
		if err == nil {
			err = s.session.Transcribed(id, t)
		}
		if err != nil {
			s.session.TranscriptionFailed(id)
			s.finish(err)
			return
		}
		clean()
		s.finish(s.review(ctx, id, reviewKey, model))
	}()
	return nil
}

// RetryReview repeats only the review of the transcript already obtained.
func (s *Service) RetryReview(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("processamento em andamento")
	}
	key, err := s.store.Get("deepseek")
	if err != nil {
		return err
	}
	if key == "" {
		return errors.New("configure as chaves de API antes de iniciar")
	}
	if err = s.session.RetryReview(id); err != nil {
		return err
	}
	model := s.model
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	s.cancel = cancel
	s.busy = true
	s.message = ""
	go func() {
		defer cancel()
		s.finish(s.review(ctx, id, key, model))
	}()
	return nil
}

// review sends the transcript in batches and publishes the proposal only when every batch is complete.
func (s *Service) review(ctx context.Context, id uint64, key, model string) error {
	s.report("reviewing", 0)
	batches, err := editorial.Batches(s.session.Snapshot().Original, 12000)
	if err != nil {
		_ = s.session.Reviewed(id, "", false)
		return err
	}
	var parts []string
	var changes []editorial.Change
	var revisions []providers.Revision
	for i, batch := range batches {
		result, err := s.client.Review(ctx, key, model, batch, true)
		if err != nil {
			_ = s.session.Reviewed(id, "", false)
			return err
		}
		s.report("reviewing", float64(i+1)/float64(len(batches)))
		changes = append(changes, result.Changes...)
		revisions = append(revisions, result)
		for _, seg := range result.Segments {
			parts = append(parts, strings.TrimSpace(seg.Text))
		}
	}
	if err = s.session.Reviewed(id, strings.Join(parts, "\n\n"), true); err != nil {
		return err
	}
	s.mu.Lock()
	s.changes = changes
	s.revisions = revisions
	s.mu.Unlock()
	return nil
}
func (s *Service) Cancel(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.session.Cancel(id); err != nil {
		return err
	}
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
func (s *Service) DeleteRemote(id uint64, confirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy || id != s.session.Snapshot().ID || !confirmed || s.remoteID == "" {
		return errors.New("exclusão do registro não confirmada ou indisponível")
	}
	key, err := s.store.Get("assemblyai")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err = s.client.DeleteTranscript(ctx, key, s.remoteID); err != nil {
		return err
	}
	s.remoteDeleted = true
	return nil
}
func (s *Service) finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	s.cancel = nil
	s.stage, s.progress = "", 0
	if err != nil {
		s.message = describe(err)
	}
}

// Shutdown cancels subprocesses and HTTP requests and waits for deferred temporary cleanup.
func (s *Service) Shutdown() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	for {
		s.mu.Lock()
		busy := s.busy
		s.mu.Unlock()
		if !busy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *Service) RecoverTemporaries() {
	if err := media.Recover(); err != nil {
		s.mu.Lock()
		s.message = "Não foi possível recuperar temporários; confira permissões do cache antes de processar mídia."
		s.mu.Unlock()
	}
}
