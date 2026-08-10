package domain

import "testing"

func TestEvaluateAnswer(t *testing.T) {
	tests := []struct {
		name, reply, target, kind string
		correct                   bool
		distance                  int
	}{
		{"exact normalized", "  HOLA ", "hola", AnswerExact, true, 0},
		{"short typo", "hol", "hola", AnswerTypo, false, 1},
		{"long typo", "restaurate", "restaurante", AnswerTypo, false, 1},
		{"wrong", "adios", "hola", AnswerWrong, false, 5},
		{"skip", "", "hola", AnswerSkipped, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateAnswer(tt.reply, tt.target)
			if got.Kind != tt.kind || got.Correct != tt.correct || got.Distance != tt.distance {
				t.Fatalf("EvaluateAnswer() = %#v", got)
			}
		})
	}
}
