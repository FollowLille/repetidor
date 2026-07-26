package storage

import (
	"context"
	"repetidor/internal/domain"
)

type TrainingRepository interface {
	Save(ctx context.Context, wordID int64, direction string, question string, expected string, response string, ok bool) error
	ListProgress(ctx context.Context, direction string) (map[int64]domain.TrainingProgress, error)
	ListStats(ctx context.Context) ([]domain.TrainingWordStats, error)
}
