package game

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestChoicesIncludesTargetWithoutDuplicates(t *testing.T) {
	got := Choices("pan", []string{"carne", "pan", "agua", "agua", "leche"}, 4, rand.New(rand.NewSource(7)))
	if len(got) != 4 {
		t.Fatalf("Choices() length = %d, want 4: %v", len(got), got)
	}
	seen := map[string]bool{}
	for _, choice := range got {
		if seen[choice] {
			t.Fatalf("Choices() contains duplicate %q: %v", choice, got)
		}
		seen[choice] = true
	}
	if !seen["pan"] {
		t.Fatalf("Choices() omitted target: %v", got)
	}
}

func TestMaskWordKeepsEdgesAndSeparators(t *testing.T) {
	if got, want := MaskWord("buenos días"), "b••••s d••s"; got != want {
		t.Fatalf("MaskWord() = %q, want %q", got, want)
	}
	if got := MaskWord("pan"); got != "pan" {
		t.Fatalf("short MaskWord() = %q, want original", got)
	}
}

func TestShuffledLettersPreservesRunes(t *testing.T) {
	got := ShuffledLetters("niño", rand.New(rand.NewSource(3)))
	if strings.Join(got, "") == "niño" {
		t.Fatalf("ShuffledLetters() did not shuffle: %v", got)
	}
	counts := func(items []string) map[string]int {
		out := map[string]int{}
		for _, item := range items {
			out[item]++
		}
		return out
	}
	if !reflect.DeepEqual(counts(got), counts([]string{"n", "i", "ñ", "o"})) {
		t.Fatalf("ShuffledLetters() changed letters: %v", got)
	}
}

func TestArenaModesAreDiscoverable(t *testing.T) {
	for _, slug := range []string{"choice", "cloze", "anagram", "match"} {
		if mode, ok := FindMode(slug); !ok || mode.Name == "" {
			t.Fatalf("FindMode(%q) = %#v, %v", slug, mode, ok)
		}
	}
}
