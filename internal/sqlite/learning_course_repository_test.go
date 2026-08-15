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
	basics, err := courses.Create(ctx, domain.LearningCourse{LanguageTrackID: 1, Name: "Foundations", SortOrder: 1, TopicIDs: []int64{verbs.ID}})
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
	if restaurant.LanguageTrackID != 1 {
		t.Fatalf("track = %d", restaurant.LanguageTrackID)
	}
}
