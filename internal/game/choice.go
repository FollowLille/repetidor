package game

import "math/rand"

func Choices(target string, candidates []string, size int, rng *rand.Rand) []string {
	if size < 2 {
		size = 2
	}
	unique := []string{target}
	seen := map[string]bool{target: true}
	shuffled := append([]string(nil), candidates...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, candidate := range shuffled {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		unique = append(unique, candidate)
		if len(unique) == size {
			break
		}
	}
	rng.Shuffle(len(unique), func(i, j int) { unique[i], unique[j] = unique[j], unique[i] })
	return unique
}
