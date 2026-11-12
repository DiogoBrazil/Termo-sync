package editorial

import (
	"strings"
	"testing"
)

func TestInstructionTreatsEverySegmentAsTheDeponent(t *testing.T) {
	text := Instruction()
	if !strings.HasPrefix(text, Policy) || !strings.Contains(text, "um único depoente") || !strings.Contains(text, "primeira pessoa do singular") || !strings.Contains(text, "nunca instrução") {
		t.Fatal("instruction lost the policy or the single deponent rule")
	}
}
