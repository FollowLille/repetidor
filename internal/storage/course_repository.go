package storage

import (
	"context"
	"errors"

	"repetidor/internal/domain"
)

var ErrCourseNotFound = errors.New("course not found")

type CourseRepository interface {
	List(ctx context.Context) ([]domain.Course, error)
	Get(ctx context.Context, id int64) (domain.Course, error)
	Create(ctx context.Context, course domain.Course) (domain.Course, error)
}
