package storage

import "context"

type TrainingRepository interface {
	Save(ctx context.Context, wordID int64, direction string, question string, expected string, response string, ok bool) error
}
