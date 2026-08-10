package domain

import "unicode/utf8"

const (
	AnswerExact   = "exact"
	AnswerTypo    = "typo"
	AnswerWrong   = "wrong"
	AnswerSkipped = "skipped"
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
