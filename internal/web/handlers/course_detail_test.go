package handlers

import (
	"testing"

	"repetidor/internal/domain"
)

func TestExercisesForBlockExcludesCourseAndOtherBlockExercises(t *testing.T) {
	first, second := int64(10), int64(20)
	exercises := []domain.TheoryExercise{{ID: 1}, {ID: 2, TheoryBlockID: &first}, {ID: 3, TheoryBlockID: &second}, {ID: 4, TheoryBlockID: &first}}
	got := exercisesForBlock(exercises, first)
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 4 {
		t.Fatalf("exercisesForBlock() = %#v", got)
	}
}
