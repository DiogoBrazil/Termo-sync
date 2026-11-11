// Package application owns a single in-memory document session. It never persists content.
package application

import (
	"errors"
	"sync"
	"termo-sync/internal/domain"
)

type State string

const (
	AwaitConsent       State = "awaiting_consent"
	AwaitTrack         State = "awaiting_track"
	Transcribing       State = "transcribing"
	AwaitSpeaker       State = "awaiting_speaker"
	AwaitReviewConsent State = "awaiting_review_consent"
	Reviewing          State = "reviewing"
	Ready              State = "ready"
	Failed             State = "failed"
	Cancelled          State = "cancelled"
)

type Snapshot struct {
	ID       uint64
	File     string
	State    State
	Track    int
	Speaker  string
	Original domain.Transcript
	Proposal string
	Edited   string
	HasEdits bool
}

type Session struct {
	mu     sync.Mutex
	next   uint64
	doc    Snapshot
	tracks []int
}

// Open invalidates all results and authorizations belonging to the preceding file.
func (s *Session) Open(file string, tracks []int) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if file == "" || len(tracks) == 0 {
		return 0, errors.New("arquivo e trilhas de áudio obrigatórios")
	}
	s.next++
	s.tracks = append([]int(nil), tracks...)
	s.doc = Snapshot{ID: s.next, File: file, State: AwaitConsent, Track: tracks[0]}
	if len(tracks) > 1 {
		s.doc.State = AwaitTrack
	}
	return s.next, nil
}
func (s *Session) SelectTrack(id uint64, track int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, AwaitTrack) {
		return errors.New("seleção fora de etapa")
	}
	for _, v := range s.tracks {
		if v == track {
			s.doc.Track = track
			s.doc.State = AwaitConsent
			return nil
		}
	}
	return errors.New("trilha inexistente")
}

// Authorize accepts only the provider required by the current stage, per file.
func (s *Session) Authorize(id uint64, provider string, confirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !confirmed {
		return errors.New("envio não autorizado")
	}
	switch {
	case provider == "assemblyai" && s.is(id, AwaitConsent):
		s.doc.State = Transcribing
	case provider == "deepseek" && s.is(id, AwaitReviewConsent):
		s.doc.State = Reviewing
	default:
		return errors.New("autorização fora de etapa")
	}
	return nil
}
func (s *Session) Transcribed(id uint64, t domain.Transcript) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Transcribing) {
		return errors.New("resultado antigo ou fora de etapa")
	}
	if t.Status != "completed" || t.Text == "" || len(t.Utterances) == 0 {
		return errors.New("transcrição incompleta ou sem falantes")
	}
	s.doc.Original = clone(t)
	s.doc.State = AwaitSpeaker
	return nil
}
func (s *Session) Identify(id uint64, speaker string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, AwaitSpeaker) {
		return errors.New("identificação fora de etapa")
	}
	for _, u := range s.doc.Original.Utterances {
		if u.Speaker == speaker && speaker != "" {
			s.doc.Speaker = speaker
			s.doc.State = AwaitReviewConsent
			return nil
		}
	}
	return errors.New("falante inexistente")
}
func (s *Session) Reviewed(id uint64, proposal string, complete bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Reviewing) {
		return errors.New("resultado antigo ou fora de etapa")
	}
	if !complete || proposal == "" {
		s.doc.State = Failed
		return errors.New("revisão incompleta")
	}
	s.doc.Proposal = proposal
	s.doc.State = Ready
	return nil
}
func (s *Session) Edit(id uint64, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Ready) {
		return errors.New("edição fora de etapa")
	}
	s.doc.Edited = text
	s.doc.HasEdits = true
	return nil
}

// RetryReview keeps the original and requires a new explicit review authorization.
func (s *Session) RetryReview(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Failed) || s.doc.Speaker == "" {
		return errors.New("revisão indisponível")
	}
	s.doc.State = AwaitReviewConsent
	return nil
}
func (s *Session) Cancel(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.ID != id || id == 0 {
		return errors.New("sessão antiga")
	}
	s.doc.State = Cancelled
	return nil
}
func (s *Session) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.doc
	d.Original = clone(d.Original)
	return d
}
func (s *Session) is(id uint64, state State) bool {
	return id != 0 && s.doc.ID == id && s.doc.State == state
}
func clone(t domain.Transcript) domain.Transcript {
	t.Raw = append([]byte(nil), t.Raw...)
	t.Utterances = append([]domain.Utterance(nil), t.Utterances...)
	return t
}
