package storage

import (
	"context"

	"repetidor/internal/domain"
)

type TheoryRepository interface {
	ListBlocks(ctx context.Context, courseID int64) ([]domain.TheoryBlock, error)
	CreateBlock(ctx context.Context, block domain.TheoryBlock) (domain.TheoryBlock, error)
	DeleteBlock(ctx context.Context, id int64) error
	ListExercises(ctx context.Context, courseID int64) ([]domain.TheoryExercise, error)
	CreateExercise(ctx context.Context, exercise domain.TheoryExercise) (domain.TheoryExercise, error)
	DeleteExercise(ctx context.Context, id int64) error
	Progress(ctx context.Context, courseID int64) (domain.CourseProgress, error)
	MarkTheoryRead(ctx context.Context, courseID int64) (domain.CourseProgress, error)
	SubmitAnswer(ctx context.Context, exerciseID int64, answer string) (domain.TheoryAnswerResult, error)
}
