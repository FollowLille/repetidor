package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"repetidor/internal/domain"
)

func TestLearningCoursesFormTreeAndShareTopics(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "learning-courses.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	topics := NewTopicRepository(db)
	comida, err := topics.Create(ctx, domain.Topic{CourseID: 1, Name: "Comida"})
	if err != nil {
		t.Fatal(err)
	}
	verbs, err := topics.Create(ctx, domain.Topic{CourseID: 1, Name: "Verbs"})
	if err != nil {
		t.Fatal(err)
	}
	courses := NewLearningCourseRepository(db)
	levels, err := courses.ListLevels(ctx, 1)
	if err != nil || len(levels) != 6 || levels[0].Code != "A1" || levels[5].Code != "C2" {
		t.Fatalf("levels = %#v, err = %v", levels, err)
	}
	basics, err := courses.Create(ctx, domain.LearningCourse{LanguageTrackID: 1, Name: "Foundations", SortOrder: 1, TopicIDs: []int64{verbs.ID}, LevelIDs: []int64{levels[0].ID, levels[1].ID}})
	if err != nil {
		t.Fatal(err)
	}
	parent := basics.ID
	restaurant, err := courses.Create(ctx, domain.LearningCourse{LanguageTrackID: 1, ParentID: &parent, Name: "At the restaurant", SortOrder: 2, TopicIDs: []int64{comida.ID, verbs.ID}, PrerequisiteIDs: []int64{basics.ID}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := courses.ListByTrack(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d courses, want 2", len(got))
	}
	if got[1].ParentID == nil || *got[1].ParentID != basics.ID {
		t.Fatalf("child parent = %#v", got[1].ParentID)
	}
	if len(got[1].TopicIDs) != 2 || got[1].TopicIDs[1] != verbs.ID {
		t.Fatalf("shared topics = %#v", got[1].TopicIDs)
	}
	if len(got[1].PrerequisiteIDs) != 1 || got[1].PrerequisiteIDs[0] != basics.ID {
		t.Fatalf("prerequisites = %#v", got[1].PrerequisiteIDs)
	}
	if len(got[0].LevelIDs) != 2 || got[0].LevelIDs[0] != levels[0].ID {
		t.Fatalf("course levels = %#v", got[0].LevelIDs)
	}
	if restaurant.LanguageTrackID != 1 {
		t.Fatalf("track = %d", restaurant.LanguageTrackID)
	}
	restaurant.Name = "Restaurant practice"
	restaurant.TopicIDs = []int64{comida.ID}
	restaurant.PrerequisiteIDs = nil
	restaurant.LevelIDs = []int64{levels[1].ID}
	if _, err := courses.Update(ctx, restaurant); err != nil {
		t.Fatal(err)
	}
	updated, err := courses.Get(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Restaurant practice" || len(updated.TopicIDs) != 1 || updated.TopicIDs[0] != comida.ID || len(updated.PrerequisiteIDs) != 0 || len(updated.LevelIDs) != 1 || updated.LevelIDs[0] != levels[1].ID {
		t.Fatalf("updated course = %#v", updated)
	}
}
