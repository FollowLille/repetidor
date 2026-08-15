package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"repetidor/internal/domain"
)

func TestCoursesAndTopicsAreSeparated(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "courses.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	courses := NewCourseRepository(db)
	topics := NewTopicRepository(db)
	defaultCourse, err := courses.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if defaultCourse.TargetLanguage != "es" || defaultCourse.ReferenceLanguage != "ru" {
		t.Fatalf("default course = %#v", defaultCourse)
	}
	english, err := courses.Create(ctx, domain.Course{Name: "English", TargetLanguage: "en", ReferenceLanguage: "ru", TheoryLanguage: "ru"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := topics.Create(ctx, domain.Topic{CourseID: 1, Name: "Comida"}); err != nil {
		t.Fatal(err)
	}
	if _, err := topics.Create(ctx, domain.Topic{CourseID: english.ID, Name: "Basics"}); err != nil {
		t.Fatal(err)
	}
	spanishTopics, err := topics.ListByCourse(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	englishTopics, err := topics.ListByCourse(ctx, english.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(spanishTopics) != 1 || spanishTopics[0].Name != "Comida" || len(englishTopics) != 1 || englishTopics[0].Name != "Basics" {
		t.Fatalf("topics leaked between courses: %#v / %#v", spanishTopics, englishTopics)
	}
}
