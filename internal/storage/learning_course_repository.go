package storage

import (
	"context"
	"errors"

	"repetidor/internal/domain"
)

var ErrLearningCourseNotFound = errors.New("learning course not found")

type LearningCourseRepository interface {
	ListByTrack(ctx context.Context, trackID int64) ([]domain.LearningCourse, error)
	Create(ctx context.Context, course domain.LearningCourse) (domain.LearningCourse, error)
}
