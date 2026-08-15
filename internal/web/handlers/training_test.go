package handlers

import (
	"math/rand"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"repetidor/internal/domain"
)

func TestTopicFiltersAcceptsMultipleUniqueTopics(t *testing.T) {
	r := httptest.NewRequest("GET", "/train/mixed?topic_ids=7&topic_ids=3&topic_ids=7&topic_ids=bad", nil)
	got := topicFilters(r)
	if len(got) != 2 || got[0] != 7 || got[1] != 3 {
		t.Fatalf("topicFilters() = %v", got)
	}
}

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

func TestTrainingPathPreservesTopicFilter(t *testing.T) {
	got := trainingPath("mixed", 7)
	want := "/train/mixed?topic_id=7"
	if got != want {
		t.Fatalf("trainingPath() = %q, want %q", got, want)
	}
}

func TestSessionURL(t *testing.T) {
	got := sessionURL("type", 13)
	want := "/train/type?session_id=13"
	if got != want {
		t.Fatalf("sessionURL() = %q, want %q", got, want)
	}
}

func TestArenaModesKeepTheirAnswerStyle(t *testing.T) {
	for _, mode := range []string{"choice", "cloze", "anagram", "match"} {
		if got := cleanMode(mode); got != mode {
			t.Errorf("cleanMode(%q) = %q", mode, got)
		}
		if got := cleanAnswerMode("", mode); got != mode {
			t.Errorf("cleanAnswerMode('', %q) = %q", mode, got)
		}
	}
}

func TestArenaGameModesAreCleanAndStable(t *testing.T) {
	got := arenaGameModes([]string{"cloze", "choice", "cloze", "bad", "match"})
	want := []string{"cloze", "choice", "match"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("arenaGameModes() = %v, want %v", got, want)
	}
	if got := arenaGameModes(nil); !reflect.DeepEqual(got, []string{"choice", "cloze", "anagram", "match"}) {
		t.Fatalf("default arenaGameModes() = %v", got)
	}
}

func TestCustomArenaCyclesSelectedGames(t *testing.T) {
	cards := []trainingCard{
		{Word: domain.Word{ID: 1}, Topic: domain.Topic{ID: 1}},
		{Word: domain.Word{ID: 2}, Topic: domain.Topic{ID: 1}},
	}
	progress := map[string]map[int64]domain.TrainingProgress{"spanish_to_russian": {}, "russian_to_spanish": {}}
	queue := buildSessionQueue("arcade", cards, progress, 5, "spanish_to_russian", "both", "choice,anagram,cloze")
	want := []string{"choice", "anagram", "cloze", "choice", "anagram"}
	for i, card := range queue {
		if card.AnswerMode != want[i] {
			t.Fatalf("queue[%d].AnswerMode = %q, want %q", i, card.AnswerMode, want[i])
		}
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

func TestBuildSessionQueueBalancesTopicsWordsAndDirections(t *testing.T) {
	cards := []trainingCard{
		{Word: domain.Word{ID: 1}, Topic: domain.Topic{ID: 10}},
		{Word: domain.Word{ID: 2}, Topic: domain.Topic{ID: 10}},
		{Word: domain.Word{ID: 3}, Topic: domain.Topic{ID: 20}},
		{Word: domain.Word{ID: 4}, Topic: domain.Topic{ID: 20}},
	}
	progress := map[string]map[int64]domain.TrainingProgress{"spanish_to_russian": {}, "russian_to_spanish": {}}
	queue := buildSessionQueue("mixed", cards, progress, 8, "spanish_to_russian", "both", "both")
	if len(queue) != 8 {
		t.Fatalf("queue length = %d", len(queue))
	}
	topicCounts := map[int64]int{}
	for i, card := range queue {
		topicCounts[card.TopicID]++
		if i > 0 && card.WordID == queue[i-1].WordID {
			t.Fatalf("adjacent duplicate at %d: %#v", i, queue)
		}
		wantDirection := "spanish_to_russian"
		wantAnswerMode := "type"
		if i%2 == 1 {
			wantDirection = "russian_to_spanish"
			wantAnswerMode = "build"
		}
		if card.Direction != wantDirection || card.AnswerMode != wantAnswerMode {
			t.Fatalf("card %d = %#v", i, card)
		}
	}
	if topicCounts[10] != 4 || topicCounts[20] != 4 {
		t.Fatalf("topic counts = %v", topicCounts)
	}
}

func TestRandomSelectionIsUniform(t *testing.T) {
	cards := []trainingCard{{Word: domain.Word{ID: 1}}, {Word: domain.Word{ID: 2}}, {Word: domain.Word{ID: 3}}}
	rng := rand.New(rand.NewSource(42))
	counts := map[int64]int{}
	for i := 0; i < 30000; i++ {
		counts[pickCardForModeWithRand("random", cards, nil, rng).Word.ID]++
	}
	for id := int64(1); id <= 3; id++ {
		if counts[id] < 9700 || counts[id] > 10300 {
			t.Fatalf("random distribution = %v", counts)
		}
	}
}

func TestMixedSelectionUsesRecentPain(t *testing.T) {
	cards := []trainingCard{{Word: domain.Word{ID: 1}}, {Word: domain.Word{ID: 2}}}
	progress := map[int64]domain.TrainingProgress{2: {WordID: 2, SeenCount: 1, RecentPain: 8}}
	rng := rand.New(rand.NewSource(42))
	counts := map[int64]int{}
	for i := 0; i < 10000; i++ {
		counts[pickCardForModeWithRand("mixed", cards, progress, rng).Word.ID]++
	}
	if counts[2] < counts[1]*4 {
		t.Fatalf("mixed did not prioritize pain: %v", counts)
	}
}

func TestFilterCardsForDirectionsIncludesPainFromEitherDirection(t *testing.T) {
	cards := []trainingCard{{Word: domain.Word{ID: 1}}, {Word: domain.Word{ID: 2}}, {Word: domain.Word{ID: 3}}}
	progress := map[string]map[int64]domain.TrainingProgress{
		"spanish_to_russian": {1: {WordID: 1, RecentPain: 2}},
		"russian_to_spanish": {2: {WordID: 2, RecentPain: 1}},
	}
	assertCardIDs(t, filterCardsForDirections("hard", cards, progress, time.Now()), []int64{1, 2})
}

func TestLanguageAgnosticDirectionModes(t *testing.T) {
	if got := cleanMode("target-to-reference"); got != "target-to-reference" {
		t.Fatalf("forward mode = %q", got)
	}
	if got := cleanDirectionMode("", "target-to-reference"); got != "spanish_to_russian" {
		t.Fatalf("forward direction = %q", got)
	}
	if got := cleanDirectionMode("", "reference-to-target"); got != "russian_to_spanish" {
		t.Fatalf("reverse direction = %q", got)
	}
	course := domain.Course{TargetLanguage: "en", ReferenceLanguage: "ru"}
	if got := labelDirection("spanish_to_russian", course); got != "English → Русский" {
		t.Fatalf("forward label = %q", got)
	}
	if got := labelDirection("russian_to_spanish", course); got != "Русский → English" {
		t.Fatalf("reverse label = %q", got)
	}
}
