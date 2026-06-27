package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type TrainingRepository struct {
	db *sql.DB
}

func NewTrainingRepository(db *sql.DB) *TrainingRepository {
	return &TrainingRepository{db: db}
}

func (r *TrainingRepository) Save(ctx context.Context, wordID int64, direction string, question string, expected string, response string, ok bool) error {
	correct := 0
	if ok {
		correct = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO training_attempts(word_id, direction, question, expected, response, is_correct)
		VALUES(?, ?, ?, ?, ?, ?)
	`, wordID, direction, question, expected, response, correct)
	if err != nil {
		return fmt.Errorf("save training attempt: %w", err)
	}
	return nil
}
