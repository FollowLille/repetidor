package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"repetidor/internal/domain"
)

func TestTheoryPracticeUpdatesCourseProgressAtomically(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "theory.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	courses := NewLearningCourseRepository(db)
	course, err := courses.Create(ctx, domain.LearningCourse{LanguageTrackID: 1, Name: "Present tense"})
	if err != nil {
		t.Fatal(err)
	}
	theory := NewTheoryRepository(db)
	block, err := theory.CreateBlock(ctx, domain.TheoryBlock{CourseID: course.ID, Kind: "example", Title: "-ar verbs", Content: "hablar → hablo", SortOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	exercise, err := theory.CreateExercise(ctx, domain.TheoryExercise{CourseID: course.ID, TheoryBlockID: &block.ID, Kind: "input", Prompt: "Yo ___ español", CorrectAnswer: "hablo", Explanation: "Use the first-person ending -o."})
	if err != nil {
		t.Fatal(err)
	}
	progress, err := theory.MarkTheoryRead(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.TheoryReadAt == nil || progress.Percent() != 50 {
		t.Fatalf("read progress = %#v, %d%%", progress, progress.Percent())
	}
	result, err := theory.SubmitAnswer(ctx, exercise.ID, " HABLO ")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Correct {
		t.Fatalf("answer should be correct: %#v", result)
	}
	if result.Progress.PracticeCompletedAt == nil || result.Progress.PracticeCorrect != 1 || result.Progress.PracticeTotal != 1 || result.Progress.Percent() != 100 {
		t.Fatalf("completed progress = %#v, %d%%", result.Progress, result.Progress.Percent())
	}
	var attempts int
	if err := db.QueryRow(`SELECT COUNT(*) FROM theory_attempts WHERE exercise_id=?`, exercise.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestTheoryWrongAnswerKeepsFeedbackAndProgress(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "wrong-theory.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	course, err := NewLearningCourseRepository(db).Create(ctx, domain.LearningCourse{LanguageTrackID: 1, Name: "Pronouns"})
	if err != nil {
		t.Fatal(err)
	}
	theory := NewTheoryRepository(db)
	exercise, err := theory.CreateExercise(ctx, domain.TheoryExercise{CourseID: course.ID, Kind: "choice", Prompt: "I", Options: []string{"yo", "tú"}, CorrectAnswer: "yo", Explanation: "Yo is first person singular."})
	if err != nil {
		t.Fatal(err)
	}
	result, err := theory.SubmitAnswer(ctx, exercise.ID, "tú")
	if err != nil {
		t.Fatal(err)
	}
	if result.Correct || result.Expected != "yo" || result.Explanation == "" {
		t.Fatalf("wrong feedback = %#v", result)
	}
	if result.Progress.PracticeCorrect != 0 || result.Progress.PracticeTotal != 1 {
		t.Fatalf("wrong progress = %#v", result.Progress)
	}
}
