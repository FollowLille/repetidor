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

func TestCoursePackageChangedImportUpdatesStableIDsAndKeepsAttempts(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "updates.sqlite3"))
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
	value.Course.Topics[0].Words[0].Key = "bread"
	courseID, _, err := store.Import(ctx, 1, value)
	if err != nil {
		t.Fatal(err)
	}
	var blockID, exerciseID int64
	if err = db.QueryRow(`SELECT id FROM theory_blocks WHERE course_id=?`, courseID).Scan(&blockID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT id FROM theory_exercises WHERE course_id=?`, courseID).Scan(&exerciseID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO theory_attempts(exercise_id,answer,is_correct) VALUES(?,?,1)`, exerciseID, "soy"); err != nil {
		t.Fatal(err)
	}
	updated := value
	updated.Course.Description = "Updated"
	updated.Course.Blocks[0].Content = "Updated rule"
	updated.Course.Exercises[0].Prompt = "Yo ___ español"
	updated.Course.Exercises = append(updated.Course.Exercises, coursepack.Exercise{Key: "ser-2", BlockKey: "ser", Kind: "input", Prompt: "Tú ___", CorrectAnswer: "eres", SortOrder: 2})
	updatedID, _, err := store.Import(ctx, 1, updated)
	if err != nil {
		t.Fatal(err)
	}
	if updatedID != courseID {
		t.Fatalf("course ID changed: %d -> %d", courseID, updatedID)
	}
	var gotBlockID, gotExerciseID, exerciseCount, attemptCount int64
	_ = db.QueryRow(`SELECT id FROM theory_blocks WHERE course_id=?`, courseID).Scan(&gotBlockID)
	_ = db.QueryRow(`SELECT id FROM theory_exercises WHERE course_id=? AND prompt=?`, courseID, "Yo ___ español").Scan(&gotExerciseID)
	_ = db.QueryRow(`SELECT COUNT(*) FROM theory_exercises WHERE course_id=?`, courseID).Scan(&exerciseCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM theory_attempts WHERE exercise_id=?`, exerciseID).Scan(&attemptCount)
	if gotBlockID != blockID || gotExerciseID != exerciseID || exerciseCount != 2 || attemptCount != 1 {
		t.Fatalf("IDs/statistics changed: block=%d/%d exercise=%d/%d count=%d attempts=%d", blockID, gotBlockID, exerciseID, gotExerciseID, exerciseCount, attemptCount)
	}
	exported, err := store.Export(ctx, courseID)
	if err != nil {
		t.Fatal(err)
	}
	wordKeyFound := false
	for _, word := range exported.Course.Topics[0].Words {
		wordKeyFound = wordKeyFound || word.Key == "bread"
	}
	if exported.Course.Blocks[0].Key != "ser" || exported.Course.Exercises[0].Key != "ser-1" || !wordKeyFound {
		t.Fatalf("stable keys lost: %#v", exported)
	}
}

func TestCoursePackagePreviewRejectsUnknownLevelAndExerciseKind(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "validation.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	value := testCoursePackage()
	value.Course.Levels = []string{"A3"}
	if _, err = NewCoursePackageStore(db).Preview(context.Background(), 1, value); err == nil {
		t.Fatal("preview accepted unknown level")
	}
	value = testCoursePackage()
	value.Course.Exercises[0].Kind = "mystery"
	if _, err = NewCoursePackageStore(db).Preview(context.Background(), 1, value); err == nil {
		t.Fatal("preview accepted unknown exercise kind")
	}
}

func TestCoursePackageSharedVocabularyRoundTrip(t *testing.T) {
	ctx := context.Background()
	sourceDB, err := Open(filepath.Join(t.TempDir(), "shared-source.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer sourceDB.Close()
	if err = Migrate(sourceDB, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	value := testCoursePackage()
	shared := coursepack.Word{Key: "run", Target: "correr", Reference: "бегать", Notes: "verb"}
	value.Course.Topics = []coursepack.Topic{
		{Key: "sport", Name: "Sport", Words: []coursepack.Word{shared}},
		{Key: "common-actions", Name: "Common actions", Words: []coursepack.Word{shared}},
	}
	sourceStore := NewCoursePackageStore(sourceDB)
	courseID, _, err := sourceStore.Import(ctx, 1, value)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := sourceStore.Export(ctx, courseID)
	if err != nil {
		t.Fatal(err)
	}
	if err = exported.Validate(); err != nil {
		t.Fatalf("exported shared vocabulary is invalid: %v", err)
	}

	targetDB, err := Open(filepath.Join(t.TempDir(), "shared-target.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer targetDB.Close()
	if err = Migrate(targetDB, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = NewCoursePackageStore(targetDB).Import(ctx, 1, exported); err != nil {
		t.Fatal(err)
	}
	var words, links int
	if err = targetDB.QueryRow(`SELECT COUNT(*) FROM words WHERE spanish='correr' AND russian='бегать'`).Scan(&words); err != nil {
		t.Fatal(err)
	}
	if err = targetDB.QueryRow(`SELECT COUNT(*) FROM word_topics wt JOIN words w ON w.id=wt.word_id WHERE w.spanish='correr' AND w.russian='бегать'`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if words != 1 || links != 2 {
		t.Fatalf("shared vocabulary roundtrip: words=%d links=%d", words, links)
	}
}

func TestCoursePackageNormalizesLevelCodesWhenLinking(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "level-normalization.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	value := testCoursePackage()
	value.Course.Levels = []string{" a1 "}
	courseID, _, err := NewCoursePackageStore(db).Import(context.Background(), 1, value)
	if err != nil {
		t.Fatal(err)
	}
	var code string
	if err = db.QueryRow(`SELECT l.code FROM learning_levels l JOIN learning_course_levels cl ON cl.level_id=l.id WHERE cl.course_id=?`, courseID).Scan(&code); err != nil {
		t.Fatal(err)
	}
	if code != "A1" {
		t.Fatalf("linked level = %q", code)
	}
}
