package editorial

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"termo-sync/internal/domain"
	"unicode/utf8"
)

type Segment struct {
	ID      string `json:"id"`
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}
type Change struct {
	ID       string   `json:"id"`
	Original string   `json:"original"`
	Revised  string   `json:"revised"`
	Flags    []string `json:"flags"`
}

// Batches never split an utterance or silently drop oversized units.
func Batches(t domain.Transcript, speaker string, narrative bool, maxRunes int) ([][]Segment, error) {
	if _, err := Input(t, speaker, narrative); err != nil {
		return nil, err
	}
	if maxRunes < 1 {
		return nil, errors.New("limite de segmentação inválido")
	}
	var batches [][]Segment
	var current []Segment
	size := 0
	for i, u := range t.Utterances {
		if narrative && u.Speaker != speaker {
			continue
		}
		n := utf8.RuneCountInString(u.Text) + 100
		if n > maxRunes {
			return nil, errors.New("segmento excede limite; divida manualmente sem perder conteúdo")
		}
		if size+n > maxRunes && len(current) > 0 {
			batches = append(batches, current)
			current = nil
			size = 0
		}
		current = append(current, Segment{ID: fmt.Sprintf("s%06d", i), Speaker: u.Speaker, Text: u.Text})
		size += n
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	if len(batches) == 0 {
		return nil, errors.New("nenhum segmento para revisar")
	}
	return batches, nil
}

var numbers = regexp.MustCompile(`\d+(?:[.,:/-]\d+)*`)
var caution = regexp.MustCompile(`(?i)\b(não|nunca|jamais|talvez|acho|parece|ouvi|disseram|falaram|contaram|boato|segundo|certeza|lembro|mas|porém|contudo|entretanto)\b`)

func Compare(input, output []Segment) ([]Change, error) {
	if len(input) != len(output) {
		return nil, errors.New("cobertura da revisão divergente")
	}
	result := make([]Change, len(input))
	for i, a := range input {
		b := output[i]
		if a.ID != b.ID || a.Speaker != b.Speaker || strings.TrimSpace(b.Text) == "" {
			return nil, errors.New("revisão alterou identificadores, falantes ou omitiu conteúdo")
		}
		c := Change{ID: a.ID, Original: a.Text, Revised: b.Text}
		if strings.Join(numbers.FindAllString(a.Text, -1), "|") != strings.Join(numbers.FindAllString(b.Text, -1), "|") {
			c.Flags = append(c.Flags, "números ou datas alterados")
		}
		if strings.ToLower(strings.Join(caution.FindAllString(a.Text, -1), "|")) != strings.ToLower(strings.Join(caution.FindAllString(b.Text, -1), "|")) {
			c.Flags = append(c.Flags, "negação, atribuição ou incerteza alterada")
		}
		if utf8.RuneCountInString(b.Text)*2 < utf8.RuneCountInString(a.Text) {
			c.Flags = append(c.Flags, "redução substancial de conteúdo")
		}
		result[i] = c
	}
	return result, nil
}
