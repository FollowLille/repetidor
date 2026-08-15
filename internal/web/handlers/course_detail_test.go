package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"repetidor/internal/domain"
)

func TestWorkspaceDefaultsToLearner(t *testing.T) {
	request := httptest.NewRequest("GET", "/courses/1", nil)
	if authorMode(request) {
		t.Fatal("workspace defaulted to author")
	}
	request.AddCookie(&http.Cookie{Name: "repetidor_workspace_mode", Value: "author"})
	if !authorMode(request) {
		t.Fatal("explicit author mode was ignored")
	}
}

func TestShuffleTokensKeepsRepeatedTokensDistinctAndChangesOrder(t *testing.T) {
	tokens := shuffleTokens([]string{"yo", "yo", "hablo"})
	if len(tokens) != 3 {
		t.Fatalf("shuffleTokens() returned %d tokens", len(tokens))
	}
	seen := map[int]bool{}
	unchanged := true
	for index, token := range tokens {
		if seen[token.ID] {
			t.Fatalf("shuffleTokens() repeated id %d", token.ID)
		}
		seen[token.ID] = true
		unchanged = unchanged && token.ID == index
	}
	if unchanged {
		t.Fatal("shuffleTokens() exposed the correct order")
	}
}

func TestExercisesForBlockExcludesCourseAndOtherBlockExercises(t *testing.T) {
	first, second := int64(10), int64(20)
	exercises := []domain.TheoryExercise{{ID: 1}, {ID: 2, TheoryBlockID: &first}, {ID: 3, TheoryBlockID: &second}, {ID: 4, TheoryBlockID: &first}}
	got := exercisesForBlock(exercises, first)
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 4 {
		t.Fatalf("exercisesForBlock() = %#v", got)
	}
}
