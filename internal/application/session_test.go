package application

import (
	"termo-sync/internal/domain"
	"testing"
)

func transcript() domain.Transcript {
	return domain.Transcript{Status: "completed", Text: "Não.", Utterances: []domain.Utterance{{Speaker: "B", Text: "Não."}}}
}
func TestWorkflowAndIsolation(t *testing.T) {
	var s Session
	id, _ := s.Open("ficticio.mp4", []int{1, 3})
	if s.Authorize(id, "assemblyai", true) == nil {
		t.Fatal("ambiguous track accepted")
	}
	if s.SelectTrack(id, 2) == nil {
		t.Fatal("unknown track")
	}
	if err := s.SelectTrack(id, 3); err != nil {
		t.Fatal(err)
	}
	if s.Authorize(id, "assemblyai", false) == nil {
		t.Fatal("missing consent")
	}
	if err := s.Authorize(id, "assemblyai", true); err != nil {
		t.Fatal(err)
	}
	original := transcript()
	if err := s.Transcribed(id, original); err != nil {
		t.Fatal(err)
	}
	original.Utterances[0].Text = "changed"
	snap := s.Snapshot()
	snap.Original.Utterances[0].Text = "also changed"
	if s.Snapshot().Original.Utterances[0].Text != "Não." {
		t.Fatal("original mutated")
	}
	if s.Authorize(id, "deepseek", true) == nil {
		t.Fatal("speaker not identified")
	}
	if err := s.Identify(id, "B"); err != nil {
		t.Fatal(err)
	}
	if err := s.Authorize(id, "deepseek", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Reviewed(id, "Não.", false); err == nil {
		t.Fatal("truncation accepted")
	}
	if err := s.RetryReview(id); err != nil {
		t.Fatal(err)
	}
	if err := s.Authorize(id, "deepseek", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Reviewed(id, "Não.", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Edit(id, ""); err != nil {
		t.Fatal(err)
	}
	if !s.Snapshot().HasEdits {
		t.Fatal("empty human edit lost")
	}
	if s.Reviewed(id, "overwrite", true) == nil {
		t.Fatal("duplicate result accepted")
	}
	next, _ := s.Open("novo.mp3", []int{0})
	if s.Transcribed(id, transcript()) == nil || next == id {
		t.Fatal("stale result accepted")
	}
	if s.Snapshot().Proposal != "" {
		t.Fatal("file data leaked")
	}
	if err := s.Authorize(next, "assemblyai", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(next); err != nil {
		t.Fatal(err)
	}
	if s.Transcribed(next, transcript()) == nil {
		t.Fatal("cancelled result accepted")
	}
}
