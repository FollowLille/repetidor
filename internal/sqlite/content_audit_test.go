package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"repetidor/internal/coursepack"
	"repetidor/internal/domain"
)

func TestSpanishFirstStepsAcceptance(t *testing.T) {
	ctx := context.Background()
	files, err := filepath.Glob(filepath.Join("..", "..", "content", "spanish-first-steps", "[0-9][0-9]-*.json"))
	if err != nil || len(files) != 8 {
		t.Fatalf("pack files=%d err=%v", len(files), err)
	}
	sort.Strings(files)
	packages := make([]coursepack.Package, 0, len(files))
	for _, file := range files {
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		value, decodeErr := coursepack.Decode(data)
		if decodeErr != nil {
			t.Fatalf("%s: %v", filepath.Base(file), decodeErr)
		}
		packages = append(packages, value)
	}

	db, err := Open(filepath.Join(t.TempDir(), "spanish-first-steps.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	store := NewCoursePackageStore(db)
	ids := map[string]int64{}
	for index, value := range packages {
		id, _, importErr := store.Import(ctx, 1, value)
		if importErr != nil {
			t.Fatalf("import %s: %v", filepath.Base(files[index]), importErr)
		}
		ids[value.Course.Key] = id
	}

	for _, value := range packages {
		courseID := ids[value.Course.Key]
		if value.Course.ParentKey != "" {
			var parentID sql.NullInt64
			if err = db.QueryRow(`SELECT parent_id FROM courses WHERE id=?`, courseID).Scan(&parentID); err != nil || !parentID.Valid || parentID.Int64 != ids[value.Course.ParentKey] {
				t.Fatalf("parent %s -> %s: id=%v err=%v", value.Course.Key, value.Course.ParentKey, parentID, err)
			}
		}
		for _, prerequisite := range value.Course.Prerequisites {
			var count int
			if err = db.QueryRow(`SELECT COUNT(*) FROM course_relations WHERE course_id=? AND related_course_id=? AND relation_type='prerequisite'`, courseID, ids[prerequisite]).Scan(&count); err != nil || count != 1 {
				t.Fatalf("prerequisite %s -> %s: count=%d err=%v", value.Course.Key, prerequisite, count, err)
			}
		}
		for _, level := range value.Course.Levels {
			var count int
			if err = db.QueryRow(`SELECT COUNT(*) FROM learning_course_levels cl JOIN learning_levels l ON l.id=cl.level_id WHERE cl.course_id=? AND l.code=?`, courseID, level).Scan(&count); err != nil || count != 1 {
				t.Fatalf("level %s/%s: count=%d err=%v", value.Course.Key, level, count, err)
			}
		}
	}

	queries := map[string]string{
		"courses":         `SELECT COUNT(*) FROM courses`,
		"blocks":          `SELECT COUNT(*) FROM theory_blocks`,
		"exercises":       `SELECT COUNT(*) FROM theory_exercises`,
		"topics":          `SELECT COUNT(*) FROM topics`,
		"unique_words":    `SELECT COUNT(*) FROM words`,
		"word_links":      `SELECT COUNT(*) FROM word_topics`,
		"block_exercises": `SELECT COUNT(*) FROM theory_exercises WHERE theory_block_id IS NOT NULL`,
	}
	counts := map[string]int{}
	for name, query := range queries {
		var count int
		if err = db.QueryRow(query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		counts[name] = count
	}
	if counts["courses"] != 8 || counts["blocks"] != 22 || counts["exercises"] != 732 || counts["topics"] != 22 || counts["unique_words"] != 863 || counts["word_links"] != 946 || counts["block_exercises"] != 732 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
	for kind, expected := range map[string]int{"input": 72, "choice": 63, "gap": 467, "sentence_builder": 130} {
		var count int
		if err = db.QueryRow(`SELECT COUNT(*) FROM theory_exercises WHERE kind=?`, kind).Scan(&count); err != nil || count != expected {
			t.Fatalf("kind %s=%d want=%d err=%v", kind, count, expected, err)
		}
	}
	theory := NewTheoryRepository(db)
	var accentExerciseID int64
	if err = db.QueryRow(`SELECT entity_id FROM course_content_keys WHERE entity_type='theory_exercise' AND content_key='es.first-steps.ex.010-sentence-basics.002'`).Scan(&accentExerciseID); err != nil {
		t.Fatal(err)
	}
	for answer, expected := range map[string]domain.AnswerStatus{"tú": domain.TheoryAnswerCorrect, "tu": domain.TheoryAnswerAcceptedWithWarning, "tù": domain.TheoryAnswerWrong} {
		result, submitErr := theory.SubmitAnswer(ctx, accentExerciseID, answer)
		if submitErr != nil || result.Status != expected {
			t.Fatalf("answer %q: status=%s want=%s err=%v", answer, result.Status, expected, submitErr)
		}
	}
	var repeatedExerciseID int64
	if err = db.QueryRow(`SELECT entity_id FROM course_content_keys WHERE entity_type='theory_exercise' AND content_key='es.first-steps.ex.150-comparisons.011'`).Scan(&repeatedExerciseID); err != nil {
		t.Fatal(err)
	}
	repeatedResult, err := theory.SubmitAnswer(ctx, repeatedExerciseID, "mi coche es más rápido que tu coche")
	if err != nil || repeatedResult.Status != domain.TheoryAnswerCorrect {
		t.Fatalf("repeated-token sentence: status=%s err=%v", repeatedResult.Status, err)
	}

	for _, value := range packages {
		id, _, importErr := store.Import(ctx, 1, value)
		if !errors.Is(importErr, ErrCoursePackageDuplicate) || id != ids[value.Course.Key] {
			t.Fatalf("duplicate import %s: id=%d err=%v", value.Course.Key, id, importErr)
		}
	}

	changed := packages[3]
	changed.Course.Description += " acceptance update"
	originalExerciseKey := changed.Course.Exercises[0].Key
	var originalExerciseID int64
	if err = db.QueryRow(`SELECT entity_id FROM course_content_keys WHERE entity_type='theory_exercise' AND content_key=?`, originalExerciseKey).Scan(&originalExerciseID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO theory_attempts(exercise_id,answer,is_correct) VALUES(?,?,1)`, originalExerciseID, changed.Course.Exercises[0].CorrectAnswer); err != nil {
		t.Fatal(err)
	}
	changed.Course.Exercises = append(changed.Course.Exercises, coursepack.Exercise{Key: "acceptance.new", BlockKey: changed.Course.Blocks[0].Key, Kind: "input", Prompt: "Acceptance prompt", CorrectAnswer: "sí", SortOrder: 999})
	updatedID, _, err := store.Import(ctx, 1, changed)
	if err != nil || updatedID != ids[changed.Course.Key] {
		t.Fatalf("changed import: id=%d err=%v", updatedID, err)
	}
	var preservedID, attempts, newCount int
	_ = db.QueryRow(`SELECT entity_id FROM course_content_keys WHERE entity_type='theory_exercise' AND content_key=?`, originalExerciseKey).Scan(&preservedID)
	_ = db.QueryRow(`SELECT COUNT(*) FROM theory_attempts WHERE exercise_id=?`, originalExerciseID).Scan(&attempts)
	_ = db.QueryRow(`SELECT COUNT(*) FROM course_content_keys WHERE entity_type='theory_exercise' AND content_key='acceptance.new'`).Scan(&newCount)
	if int(originalExerciseID) != preservedID || attempts != 1 || newCount != 1 {
		t.Fatalf("changed import lost state: original=%d preserved=%d attempts=%d new=%d", originalExerciseID, preservedID, attempts, newCount)
	}
	t.Logf("counts before update: %#v", counts)
}
