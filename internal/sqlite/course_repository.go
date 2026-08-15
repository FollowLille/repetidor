package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type CourseRepository struct{ db *sql.DB }

func NewCourseRepository(db *sql.DB) *CourseRepository { return &CourseRepository{db: db} }

func (r *CourseRepository) List(ctx context.Context) ([]domain.Course, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,target_language,reference_language,theory_language,created_at,updated_at FROM courses ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	defer rows.Close()
	var courses []domain.Course
	for rows.Next() {
		var course domain.Course
		if err := rows.Scan(&course.ID, &course.Name, &course.TargetLanguage, &course.ReferenceLanguage, &course.TheoryLanguage, &course.CreatedAt, &course.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan course: %w", err)
		}
		courses = append(courses, course)
	}
	return courses, rows.Err()
}

func (r *CourseRepository) Get(ctx context.Context, id int64) (domain.Course, error) {
	var course domain.Course
	err := r.db.QueryRowContext(ctx, `SELECT id,name,target_language,reference_language,theory_language,created_at,updated_at FROM courses WHERE id=?`, id).Scan(&course.ID, &course.Name, &course.TargetLanguage, &course.ReferenceLanguage, &course.TheoryLanguage, &course.CreatedAt, &course.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Course{}, storage.ErrCourseNotFound
	}
	if err != nil {
		return domain.Course{}, fmt.Errorf("get course: %w", err)
	}
	return course, nil
}

func (r *CourseRepository) Create(ctx context.Context, course domain.Course) (domain.Course, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO courses(name,target_language,reference_language,theory_language) VALUES(?,?,?,?) RETURNING id,name,target_language,reference_language,theory_language,created_at,updated_at`, course.Name, course.TargetLanguage, course.ReferenceLanguage, course.TheoryLanguage).Scan(&course.ID, &course.Name, &course.TargetLanguage, &course.ReferenceLanguage, &course.TheoryLanguage, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return domain.Course{}, fmt.Errorf("create course: %w", err)
	}
	return course, nil
}
