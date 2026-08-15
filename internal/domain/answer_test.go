package domain

import "testing"

func TestCheckLanguageAnswerSpanishDiacritics(t *testing.T) {
	tests := []struct {
		answer, expected string
		status           AnswerStatus
	}{
		{" NIÑO ", "niño", TheoryAnswerCorrect},
		{"nino", "niño", TheoryAnswerAcceptedWithWarning},
		{"gracías", "gracias", TheoryAnswerWrong},
		{"gracias", "gracias", TheoryAnswerCorrect},
	}
	for _, tt := range tests {
		if got := CheckLanguageAnswer("es", tt.answer, []string{tt.expected}); got != tt.status {
			t.Errorf("answer %q = %q, want %q", tt.answer, got, tt.status)
		}
	}
}

func TestCheckLanguageAnswerAlternativesAndGenericPolicy(t *testing.T) {
	if got := CheckLanguageAnswer("en", " automobile ", []string{"car", "automobile"}); got != TheoryAnswerCorrect {
		t.Fatalf("alternative = %q, want correct", got)
	}
	if got := CheckLanguageAnswer("en", "cafe", []string{"café"}); got != TheoryAnswerWrong {
		t.Fatalf("generic accent omission = %q, want wrong", got)
	}
}
