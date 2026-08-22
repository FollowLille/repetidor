package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	AnswerExact    = "exact"
	AnswerTypo     = "typo"
	AnswerWrong    = "wrong"
	AnswerSkipped  = "skipped"
	AnswerDontKnow = "dont_know"
)

type AnswerEvaluation struct {
	Kind     string
	Correct  bool
	Distance int
}

func EvaluateAnswer(reply, target string) AnswerEvaluation {
	normalizedReply, normalizedTarget := NormalizeText(reply), NormalizeText(target)
	if normalizedReply == "" {
		return AnswerEvaluation{Kind: AnswerSkipped}
	}
	if normalizedReply == normalizedTarget {
		return AnswerEvaluation{Kind: AnswerExact, Correct: true}
	}
	distance := editDistance([]rune(normalizedReply), []rune(normalizedTarget))
	limit := 1
	if utf8.RuneCountInString(normalizedTarget) >= 9 {
		limit = 2
	}
	kind := AnswerWrong
	if distance <= limit {
		kind = AnswerTypo
	}
	return AnswerEvaluation{Kind: kind, Distance: distance}
}

func editDistance(left, right []rune) int {
	previous := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, a := range left {
		current := make([]int, len(right)+1)
		current[0] = i + 1
		for j, b := range right {
			cost := 0
			if a != b {
				cost = 1
			}
			current[j+1] = min3(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func min3(a, b, c int) int {
	if a > b {
		a = b
	}
	if a > c {
		a = c
	}
	return a
}

// CheckLanguageAnswer keeps the language policy behind one boundary so that
// individual languages can gain stricter rules without changing exercises.
func CheckLanguageAnswer(language, answer string, accepted []string) AnswerStatus {
	answer = strings.TrimSpace(answer)
	for _, expected := range accepted {
		expected = strings.TrimSpace(expected)
		if strings.EqualFold(answer, expected) {
			return TheoryAnswerCorrect
		}
		if language == "es" && missingDiacriticsOnly(answer, expected) {
			return TheoryAnswerAcceptedWithWarning
		}
	}
	return TheoryAnswerWrong
}

func missingDiacriticsOnly(answer, expected string) bool {
	if !strings.EqualFold(withoutMarks(answer), withoutMarks(expected)) {
		return false
	}
	answerRunes, expectedRunes := []rune(strings.ToLower(answer)), []rune(strings.ToLower(expected))
	if len(answerRunes) != len(expectedRunes) {
		return false
	}
	missing := false
	for i, got := range answerRunes {
		want := expectedRunes[i]
		if got == want {
			continue
		}
		// A differing non-ASCII rune means the learner supplied a diacritic in
		// the wrong place/form; only omission is accepted with a warning.
		if got > unicode.MaxASCII || !strings.EqualFold(string(got), withoutMarks(string(want))) {
			return false
		}
		missing = true
	}
	return missing
}

func withoutMarks(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(value))
}
