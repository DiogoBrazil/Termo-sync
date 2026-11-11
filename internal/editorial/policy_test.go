package editorial

import (
	"strings"
	"termo-sync/internal/domain"
	"testing"
)

func TestNarrativeScope(t *testing.T) {
	doc := domain.Transcript{Utterances: []domain.Utterance{{Speaker: "A", Text: "Viu a arma?"}, {Speaker: "B", Text: "Não."}}}
	if _, err := Input(doc, "", true); err == nil {
		t.Fatal("missing identity accepted")
	}
	if _, err := Input(doc, "C", true); err == nil {
		t.Fatal("unknown identity accepted")
	}
	s, err := Input(doc, "B", true)
	if err != nil || strings.Contains(s, "arma") || !strings.Contains(s, "Não.") {
		t.Fatal("narrative crossed speaker boundary")
	}
	s, err = Input(doc, "B", false)
	if err != nil || !strings.Contains(s, "Viu a arma?") {
		t.Fatal("dialogue lost question")
	}
}
