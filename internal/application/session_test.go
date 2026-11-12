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
	if s.Begin(id) == nil {
		t.Fatal("ambiguous track accepted")
	}
	if s.SelectTrack(id, 2) == nil {
		t.Fatal("unknown track")
	}
	if err := s.SelectTrack(id, 3); err != nil {
		t.Fatal(err)
	}
	if err := s.Begin(id); err != nil {
		t.Fatal(err)
	}
	if s.Begin(id) == nil {
		t.Fatal("processing started twice")
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
	if s.Snapshot().State != Reviewing {
		t.Fatal("review must follow the transcription without another step")
	}
	if err := s.Reviewed(id, "Não.", false); err == nil {
		t.Fatal("truncation accepted")
	}
	if err := s.RetryReview(id); err != nil {
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
	if err := s.Begin(next); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(next); err != nil {
		t.Fatal(err)
	}
	if s.Transcribed(next, transcript()) == nil {
		t.Fatal("cancelled result accepted")
	}
}

func TestCancelAndFailureKeepWhatWasObtained(t *testing.T) {
	var s Session
	id, _ := s.Open("ficticio.mp3", []int{0})
	if s.Cancel(id) == nil {
		t.Fatal("cancel without running work accepted")
	}
	_ = s.Begin(id)
	s.TranscriptionFailed(id)
	if s.Snapshot().State != AwaitStart {
		t.Fatal("failed transcription left the session stuck")
	}
	if err := s.Begin(id); err != nil {
		t.Fatal("a new attempt must be possible")
	}
	if err := s.Transcribed(id, transcript()); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(id); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if snap.State != Failed || snap.Original.Text != "Não." {
		t.Fatal("cancelling the review discarded the paid transcript")
	}
	if s.Reviewed(id, "tardia", true) == nil {
		t.Fatal("late review accepted after cancel")
	}
	if err := s.RetryReview(id); err != nil {
		t.Fatal("review must be repeatable without transcribing again")
	}
}

func TestTranscriptWithoutUtterancesBecomesOneSegment(t *testing.T) {
	var s Session
	id, _ := s.Open("ficticio.mp3", []int{0})
	_ = s.Begin(id)
	if err := s.Transcribed(id, domain.Transcript{Status: "completed", Text: "Não vi."}); err != nil {
		t.Fatal(err)
	}
	if u := s.Snapshot().Original.Utterances; len(u) != 1 || u[0].Text != "Não vi." {
		t.Fatal("whole text must be reviewed as a single segment")
	}
	if s.Transcribed(id, domain.Transcript{Status: "completed"}) == nil {
		t.Fatal("empty transcript accepted")
	}
}
