package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"repetidor/internal/coursepack"
)

func testCoursePackage() coursepack.Package {
	return coursepack.Package{Format: coursepack.Format, Version: coursepack.Version, Course: coursepack.Course{Key: "spanish-a1", Name: "Spanish A1", Description: "Portable", Target: "es", Reference: "ru", Theory: "ru", Levels: []string{"A1"}, Blocks: []coursepack.Block{{Key: "ser", Kind: "text", Title: "Ser", Content: "Rule", SortOrder: 1}}, Exercises: []coursepack.Exercise{{Key: "ser-1", BlockKey: "ser", Kind: "input", Prompt: "Yo ___", CorrectAnswer: "soy", AcceptedAnswers: []string{"yo soy"}, SortOrder: 1}}, Topics: []coursepack.Topic{{Key: "food", Name: "Food", Words: []coursepack.Word{{Target: "pan", Reference: "хлеб"}, {Target: "carne", Reference: "мясо"}}}}}}
}

func TestCoursePackageImportExportAndDuplicate(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "coursepack.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	store := NewCoursePackageStore(db)
	value := testCoursePackage()
	preview, err := store.Preview(ctx, 1, value)
	if err != nil || preview.Blocks != 1 || preview.Exercises != 1 || preview.Topics != 1 || preview.Words != 2 {
		t.Fatalf("preview=%#v err=%v", preview, err)
	}
	id, summary, err := store.Import(ctx, 1, value)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 || summary.Duplicates != 0 {
		t.Fatalf("id=%d summary=%#v", id, summary)
	}
	exported, err := store.Export(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(exported.Course.Blocks) != 1 || len(exported.Course.Exercises) != 1 || exported.Course.Exercises[0].BlockKey == "" || len(exported.Course.Topics) != 1 || len(exported.Course.Topics[0].Words) != 2 {
		t.Fatalf("roundtrip lost content: %#v", exported)
	}
	duplicateID, duplicateSummary, err := store.Import(ctx, 1, value)
	if !errors.Is(err, ErrCoursePackageDuplicate) || duplicateID != id || duplicateSummary.DuplicateCourseID != id {
		t.Fatalf("duplicate id=%d summary=%#v err=%v", duplicateID, duplicateSummary, err)
	}
}

func TestCoursePackageFailedImportIsAtomic(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "atomic.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	value := testCoursePackage()
	value.Course.Key = "broken"
	value.Course.Name = "Broken"
	value.Course.Exercises[0].Kind = "unsupported"
	_, _, err = NewCoursePackageStore(db).Import(context.Background(), 1, value)
	if err == nil {
		t.Fatal("expected import error")
	}
	var count int
	if scanErr := db.QueryRow(`SELECT COUNT(*) FROM courses WHERE name='Broken'`).Scan(&count); scanErr != nil || count != 0 {
		t.Fatalf("partial course remained: count=%d err=%v", count, scanErr)
	}
}
