package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"repetidor/internal/domain"

	"github.com/go-chi/chi/v5"
)

func TestTopicNameParamDecodesEscapedUnicode(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("topic_name", "%d0%92%d0%be%d0%bf%d1%80%d0%be%d1%81%d1%8b%20%d0%b8%20%d1%81%d0%b2%d1%8f%d0%b7%d0%ba%d0%b8")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	if got := topicNameParam(request); got != "Вопросы и связки" {
		t.Fatalf("topicNameParam() = %q", got)
	}
}

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

func TestChildCoursesReturnsOnlyDirectChildren(t *testing.T) {
	rootID, otherID := int64(1), int64(2)
	courses := []domain.LearningCourse{
		{ID: 1},
		{ID: 2, ParentID: &rootID, SortOrder: 20},
		{ID: 3, ParentID: &rootID, SortOrder: 30},
		{ID: 4, ParentID: &otherID},
	}
	children := childCourses(courses, rootID)
	if len(children) != 2 || children[0].ID != 2 || children[1].ID != 3 {
		t.Fatalf("childCourses() = %#v", children)
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
