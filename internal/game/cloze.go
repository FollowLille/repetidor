package game

import "unicode"

func MaskWord(word string) string {
	runes := []rune(word)
	for start := 0; start < len(runes); {
		if !unicode.IsLetter(runes[start]) {
			start++
			continue
		}
		end := start
		for end < len(runes) && unicode.IsLetter(runes[end]) {
			end++
		}
		if end-start > 3 {
			for i := start + 1; i < end-1; i++ {
				runes[i] = '•'
			}
		}
		start = end
	}
	return string(runes)
}
