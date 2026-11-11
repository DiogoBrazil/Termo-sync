package application

import (
	"context"
	"errors"
	"fmt"
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
	Document      Snapshot
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
	return View{Document: s.session.Snapshot(), Tracks: tracks, Busy: s.busy, Error: s.message, Changes: changes, RemoteID: s.remoteID, RemoteDeleted: s.remoteDeleted}
}

// Open is called with a native-dialog path, never an arbitrary path from web content.
func (s *Service) Open(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("cancele e aguarde o trabalho atual")
	}
	if s.remoteID != "" && !s.remoteDeleted {
		return errors.New("exclua a transcrição remota ou confirme que deseja mantê-la antes de trocar de arquivo")
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
		return errors.New("confirmação de retenção remota inválida")
	}
	// This only dismisses the session reminder, never claims remote deletion.
	s.remoteID = ""
	return nil
}
func (s *Service) SelectTrack(id uint64, track int) error   { return s.session.SelectTrack(id, track) }
func (s *Service) Identify(id uint64, speaker string) error { return s.session.Identify(id, speaker) }
func (s *Service) Edit(id uint64, text string) error        { return s.session.Edit(id, text) }
func (s *Service) RetryReview(id uint64) error              { return s.session.RetryReview(id) }
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
func (s *Service) StartTranscription(id uint64, authorized, portugueseLimitationAccepted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("processamento em andamento")
	}
	if !portugueseLimitationAccepted {
		return errors.New("confirme a limitação de hesitações em português e a necessidade de conferir o áudio")
	}
	key, err := s.store.Get("assemblyai")
	if err != nil {
		return err
	}
	if err = s.session.Authorize(id, "assemblyai", authorized); err != nil {
		return err
	}
	d := s.session.Snapshot()
	info := s.info
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Hour)
	s.cancel = cancel
	s.busy = true
	s.message = ""
	go func() {
		defer cancel()
		path, clean, err := s.tools.Prepare(ctx, d.File, info, d.Track)
		defer clean()
		if err != nil {
			s.finish(err)
			return
		}
		t, err := s.client.Transcribe(ctx, key, path, true)
		s.mu.Lock()
		s.remoteID = t.ID
		s.mu.Unlock()
		if err == nil {
			err = s.session.Transcribed(id, t)
		}
		s.finish(err)
	}()
	return nil
}
func (s *Service) StartReview(id uint64, authorized, narrative bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("processamento em andamento")
	}
	d := s.session.Snapshot()
	batches, err := editorial.Batches(d.Original, d.Speaker, narrative, 12000)
	if err != nil {
		return err
	}
	key, err := s.store.Get("deepseek")
	if err != nil {
		return err
	}
	if err = s.session.Authorize(id, "deepseek", authorized); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	s.cancel = cancel
	s.busy = true
	s.message = ""
	model := s.model
	go func() {
		defer cancel()
		var text strings.Builder
		var changes []editorial.Change
		var revisions []providers.Revision
		for _, batch := range batches {
			result, err := s.client.Review(ctx, key, model, batch, true)
			if err != nil {
				_ = s.session.Reviewed(id, "", false)
				s.finish(err)
				return
			}
			changes = append(changes, result.Changes...)
			revisions = append(revisions, result)
			for _, seg := range result.Segments {
				if narrative {
					fmt.Fprintf(&text, "%s\n\n", seg.Text)
				} else {
					fmt.Fprintf(&text, "Falante %s: %s\n\n", seg.Speaker, seg.Text)
				}
			}
		}
		err := s.session.Reviewed(id, strings.TrimSpace(text.String()), true)
		if err == nil {
			s.mu.Lock()
			s.changes = changes
			s.revisions = revisions
			s.mu.Unlock()
		}
		s.finish(err)
	}()
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
		return errors.New("exclusão remota não confirmada ou indisponível")
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
	if err != nil {
		s.message = err.Error()
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
