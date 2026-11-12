package editorial

import (
	"termo-sync/internal/domain"
	"testing"
)

func TestBatchesCoverage(t *testing.T) {
	doc := domain.Transcript{Utterances: []domain.Utterance{{Speaker: "A", Text: "Viu 12 pessoas?"}, {Speaker: "B", Text: "Não. Ouvi dizer."}, {Speaker: "B", Text: "Talvez fossem duas."}}}
	batches, err := Batches(doc, 125)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 3 || batches[0][0].ID != "s000000" || batches[0][0].Speaker != "A" || batches[2][0].ID != "s000002" {
		t.Fatal("coverage lost")
	}
	if _, err := Batches(doc, 10); err == nil {
		t.Fatal("oversized unit split or lost")
	}
	input := []Segment{
		{ID: "s1", Speaker: "B", Text: "Não vi 12 pessoas em 10/05/2026. Ouvi dizer, mas não tenho certeza."},
	}
	output := []Segment{
		{ID: "s1", Speaker: "B", Text: "Vi 13 pessoas em 11/05/2026."},
	}
	changes, err := Compare(input, output)
	if err != nil || len(changes[0].Flags) < 2 {
		t.Fatal("suspicious change missed")
	}
	preserved, err := Compare(input, input)
	if err != nil || len(preserved[0].Flags) != 0 {
		t.Fatal("unchanged segment flagged")
	}
	if _, err := Compare(input, append(output, output...)); err == nil {
		t.Fatal("extra segment accepted")
	}
}
