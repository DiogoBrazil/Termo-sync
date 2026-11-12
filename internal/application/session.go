// Package application owns a single in-memory document session. It never persists content.
package application

import (
	"errors"
	"sync"
	"termo-sync/internal/domain"
)

type State string

const (
	AwaitTrack   State = "awaiting_track"
	AwaitStart   State = "awaiting_start"
	Transcribing State = "transcribing"
	Reviewing    State = "reviewing"
	Ready        State = "ready"
	// Failed means the transcript exists but the review did not finish; only the review is repeated.
	Failed State = "failed"
)

type Snapshot struct {
	ID       uint64
	File     string
	State    State
	Track    int
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

// Open invalidates all results belonging to the preceding file.
func (s *Session) Open(file string, tracks []int) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if file == "" || len(tracks) == 0 {
		return 0, errors.New("arquivo e trilhas de áudio obrigatórios")
	}
	s.next++
	s.tracks = append([]int(nil), tracks...)
	s.doc = Snapshot{ID: s.next, File: file, State: AwaitStart, Track: tracks[0]}
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
			s.doc.State = AwaitStart
			return nil
		}
	}
	return errors.New("trilha inexistente")
}

// Begin starts the single processing run of the current file. Authorization to send the content is
// given by the user outside the app, so no per-file confirmation is recorded here.
func (s *Session) Begin(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, AwaitStart) {
		return errors.New("início fora de etapa")
	}
	s.doc.State = Transcribing
	return nil
}

// Transcribed stores the immutable original and moves straight to the review. Every file holds a
// single deponent, so a transcript without utterances becomes one segment with the whole text.
func (s *Session) Transcribed(id uint64, t domain.Transcript) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Transcribing) {
		return errors.New("resultado antigo ou fora de etapa")
	}
	if t.Status != "completed" || t.Text == "" {
		return errors.New("transcrição incompleta")
	}
	s.doc.Original = clone(t)
	if len(s.doc.Original.Utterances) == 0 {
		s.doc.Original.Utterances = []domain.Utterance{{Speaker: "A", Text: t.Text}}
	}
	s.doc.State = Reviewing
	return nil
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

// RetryReview repeats only the review; the transcript already obtained is kept.
func (s *Session) RetryReview(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.is(id, Failed) || len(s.doc.Original.Utterances) == 0 {
		return errors.New("revisão indisponível")
	}
	s.doc.State = Reviewing
	return nil
}

// Cancel only interrupts running work, so late results are rejected. Nothing already obtained is
// discarded: cancelling the review keeps the transcript and leaves the review repeatable.
func (s *Session) Cancel(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.ID != id || id == 0 {
		return errors.New("sessão antiga")
	}
	switch s.doc.State {
	case Transcribing:
		s.doc.State = AwaitStart
	case Reviewing:
		s.doc.State = Failed
	default:
		return errors.New("não há processamento em andamento")
	}
	return nil
}

// TranscriptionFailed returns to the start step so the user can try again.
func (s *Session) TranscriptionFailed(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.is(id, Transcribing) {
		s.doc.State = AwaitStart
	}
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
