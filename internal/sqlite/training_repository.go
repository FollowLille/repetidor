package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"repetidor/internal/domain"
)

type TrainingRepository struct {
	db *sql.DB
}

func NewTrainingRepository(db *sql.DB) *TrainingRepository {
	return &TrainingRepository{db: db}
}

func (r *TrainingRepository) Save(ctx context.Context, wordID int64, direction string, question string, expected string, response string, ok bool) error {
	correct := 0
	wrong := 1
	if ok {
		correct = 1
		wrong = 0
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin training save transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO training_attempts(word_id, direction, question, expected, response, is_correct)
		VALUES(?, ?, ?, ?, ?, ?)
	`, wordID, direction, question, expected, response, correct)
	if err != nil {
		return fmt.Errorf("save training attempt: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO training_progress(
			word_id, direction, seen_count, correct_count, wrong_count, correct_streak, recent_pain, last_seen_at, last_correct_at, updated_at
		)
		VALUES(
			?, ?, 1, ?, ?, ?, ?, CURRENT_TIMESTAMP,
			CASE WHEN ? = 1 THEN CURRENT_TIMESTAMP ELSE NULL END,
			CURRENT_TIMESTAMP
		)
		ON CONFLICT(word_id, direction) DO UPDATE SET
			seen_count = seen_count + 1,
			correct_count = correct_count + excluded.correct_count,
			wrong_count = wrong_count + excluded.wrong_count,
			correct_streak = CASE WHEN excluded.correct_count = 1 THEN correct_streak + 1 ELSE 0 END,
			recent_pain = CASE
				WHEN excluded.correct_count = 1 THEN max(recent_pain - 1, 0)
				ELSE min(recent_pain + 2, 10)
			END,
			last_seen_at = CURRENT_TIMESTAMP,
			last_correct_at = CASE WHEN excluded.correct_count = 1 THEN CURRENT_TIMESTAMP ELSE last_correct_at END,
			updated_at = CURRENT_TIMESTAMP
	`, wordID, direction, correct, wrong, correct, wrong*2, correct)
	if err != nil {
		return fmt.Errorf("save training progress: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit training save transaction: %w", err)
	}
	return nil
}

func (r *TrainingRepository) ListProgress(ctx context.Context, direction string) (map[int64]domain.TrainingProgress, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT word_id, direction, seen_count, correct_count, wrong_count, correct_streak, recent_pain
		FROM training_progress
		WHERE direction = ?
	`, direction)
	if err != nil {
		return nil, fmt.Errorf("list training progress: %w", err)
	}
	defer rows.Close()

	progress := make(map[int64]domain.TrainingProgress)
	for rows.Next() {
		item := domain.TrainingProgress{}
		if err := rows.Scan(&item.WordID, &item.Direction, &item.SeenCount, &item.CorrectCount, &item.WrongCount, &item.CorrectStreak, &item.RecentPain); err != nil {
			return nil, fmt.Errorf("scan training progress: %w", err)
		}
		progress[item.WordID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training progress: %w", err)
	}
	return progress, nil
}
