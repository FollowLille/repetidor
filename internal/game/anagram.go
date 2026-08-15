package game

import (
	"math/rand"
	"strings"
)

func ShuffledLetters(word string, rng *rand.Rand) []string {
	letters := make([]string, 0, len(word))
	for _, r := range []rune(strings.ReplaceAll(word, " ", "")) {
		letters = append(letters, string(r))
	}
	if len(letters) < 2 {
		return letters
	}
	original := strings.Join(letters, "")
	for attempt := 0; attempt < 4; attempt++ {
		rng.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
		if strings.Join(letters, "") != original {
			break
		}
	}
	return letters
}
