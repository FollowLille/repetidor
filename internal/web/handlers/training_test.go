package handlers

import (
	"testing"
	"time"

	"repetidor/internal/domain"
)

func TestExcludeCard(t *testing.T) {
	cards := []trainingCard{
		{Word: domain.Word{ID: 1}},
		{Word: domain.Word{ID: 2}},
		{Word: domain.Word{ID: 3}},
	}
	filtered := excludeCard(cards, 2)
	if len(filtered) != 2 || filtered[0].Word.ID != 1 || filtered[1].Word.ID != 3 {
		t.Fatalf("excludeCard() = %#v", filtered)
	}
}

func TestFilterCardsForLearningModes(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Hour)
	old := now.Add(-8 * 24 * time.Hour)
	cards := []trainingCard{
		{Word: domain.Word{ID: 1}},
		{Word: domain.Word{ID: 2}},
		{Word: domain.Word{ID: 3}},
		{Word: domain.Word{ID: 4}},
	}
	progress := map[int64]domain.TrainingProgress{
		2: {WordID: 2, SeenCount: 1, CorrectStreak: 1, LastSeenAt: &recent},
		3: {WordID: 3, SeenCount: 4, RecentPain: 2, LastSeenAt: &recent},
		4: {WordID: 4, SeenCount: 5, CorrectStreak: 3, LastSeenAt: &old},
	}

	assertCardIDs(t, filterCardsForMode("due", cards, progress, now), []int64{1, 3, 4})
	assertCardIDs(t, filterCardsForMode("hard", cards, progress, now), []int64{3})
	assertCardIDs(t, filterCardsForMode("easy", cards, progress, now), []int64{4})
	assertCardIDs(t, filterCardsForMode("mixed", cards, progress, now), []int64{1, 2, 3, 4})
}

func assertCardIDs(t *testing.T, cards []trainingCard, want []int64) {
	t.Helper()
	if len(cards) != len(want) {
		t.Fatalf("card count = %d, want %d", len(cards), len(want))
	}
	for i := range want {
		if cards[i].Word.ID != want[i] {
			t.Fatalf("card %d id = %d, want %d", i, cards[i].Word.ID, want[i])
		}
	}
}

func TestSessionPathPreservesTopicFilterAndProgress(t *testing.T) {
	got := sessionPath("/train/mixed?topic_id=7", trainingSession{Size: 5, Completed: 2, Correct: 1}, 11)
	want := "/train/mixed?session_completed=2&session_correct=1&session_size=5&skip_word_id=11&topic_id=7"
	if got != want {
		t.Fatalf("sessionPath() = %q, want %q", got, want)
	}
}

func TestRepeatMistakesPathUsesUniqueMistakes(t *testing.T) {
	got := repeatMistakesPath("/train/type?topic_id=3", trainingSession{MistakeIDs: []int64{8, 13}})
	want := "/train/type?only_word_ids=8%2C13&session_completed=0&session_correct=0&session_size=2&topic_id=3"
	if got != want {
		t.Fatalf("repeatMistakesPath() = %q, want %q", got, want)
	}
}

func TestRestrictCards(t *testing.T) {
	cards := []trainingCard{
		{Word: domain.Word{ID: 1}},
		{Word: domain.Word{ID: 2}},
		{Word: domain.Word{ID: 3}},
	}
	filtered := restrictCards(cards, []int64{3, 1})
	if len(filtered) != 2 || filtered[0].Word.ID != 1 || filtered[1].Word.ID != 3 {
		t.Fatalf("restrictCards() = %#v", filtered)
	}
}
