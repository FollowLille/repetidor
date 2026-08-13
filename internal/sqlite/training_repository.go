package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"repetidor/internal/domain"
	"repetidor/internal/storage"
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

func (r *TrainingRepository) SaveSessionAnswer(ctx context.Context, sessionID int64, position int, card domain.TrainingSessionCard, question string, expected string, response string, evaluation domain.AnswerEvaluation) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session answer transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `UPDATE training_session_cards
		SET answered=1, is_correct=?, response=?, error_kind=?, edit_distance=?, answered_at=CURRENT_TIMESTAMP
		WHERE session_id=? AND position=? AND answered=0
		AND position=(SELECT completed+1 FROM training_sessions WHERE id=? AND status='active')`, evaluation.Correct, response, evaluation.Kind, evaluation.Distance, sessionID, position, sessionID)
	if err != nil {
		return fmt.Errorf("record session card answer: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return storage.ErrSessionCardNotFound
	}

	if evaluation.Kind != domain.AnswerSkipped {
		correct := 0
		wrong := 1
		if evaluation.Correct {
			correct, wrong = 1, 0
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO training_attempts(word_id,direction,question,expected,response,is_correct) VALUES(?,?,?,?,?,?)`, card.WordID, card.Direction, question, expected, response, correct); err != nil {
			return fmt.Errorf("save training attempt: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO training_progress(word_id,direction,seen_count,correct_count,wrong_count,correct_streak,recent_pain,last_seen_at,last_correct_at,updated_at)
			VALUES(?,?,1,?,?,?, ?,CURRENT_TIMESTAMP,CASE WHEN ?=1 THEN CURRENT_TIMESTAMP ELSE NULL END,CURRENT_TIMESTAMP)
			ON CONFLICT(word_id,direction) DO UPDATE SET seen_count=seen_count+1, correct_count=correct_count+excluded.correct_count,
			wrong_count=wrong_count+excluded.wrong_count, correct_streak=CASE WHEN excluded.correct_count=1 THEN correct_streak+1 ELSE 0 END,
			recent_pain=CASE WHEN excluded.correct_count=1 THEN max(recent_pain-1,0) ELSE min(recent_pain+2,10) END,
			last_seen_at=CURRENT_TIMESTAMP,last_correct_at=CASE WHEN excluded.correct_count=1 THEN CURRENT_TIMESTAMP ELSE last_correct_at END,updated_at=CURRENT_TIMESTAMP`, card.WordID, card.Direction, correct, wrong, correct, wrong*2, correct); err != nil {
			return fmt.Errorf("save training progress: %w", err)
		}
	}

	skipped := 0
	if evaluation.Kind == domain.AnswerSkipped {
		skipped = 1
	}
	if _, err := tx.ExecContext(ctx, `UPDATE training_sessions SET completed=completed+1, correct=correct+?, skipped=skipped+?,
		status=CASE WHEN completed+1>=size THEN 'completed' ELSE 'active' END,
		completed_at=CASE WHEN completed+1>=size THEN CURRENT_TIMESTAMP ELSE NULL END, updated_at=CURRENT_TIMESTAMP WHERE id=?`, evaluation.Correct, skipped, sessionID); err != nil {
		return fmt.Errorf("update training session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session answer transaction: %w", err)
	}
	return nil
}

func (r *TrainingRepository) ListProgress(ctx context.Context, direction string) (map[int64]domain.TrainingProgress, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT word_id, direction, seen_count, correct_count, wrong_count, correct_streak, recent_pain, last_seen_at
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
		var lastSeen sql.NullTime
		if err := rows.Scan(&item.WordID, &item.Direction, &item.SeenCount, &item.CorrectCount, &item.WrongCount, &item.CorrectStreak, &item.RecentPain, &lastSeen); err != nil {
			return nil, fmt.Errorf("scan training progress: %w", err)
		}
		if lastSeen.Valid {
			item.LastSeenAt = &lastSeen.Time
		}
		progress[item.WordID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training progress: %w", err)
	}
	return progress, nil
}

func (r *TrainingRepository) ListStats(ctx context.Context) ([]domain.TrainingWordStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			w.id,
			GROUP_CONCAT(t.name, ', '),
			w.spanish,
			w.russian,
			COALESCE(p.seen_count, 0),
			COALESCE(p.correct_count, 0),
			COALESCE(p.wrong_count, 0),
			COALESCE(p.correct_streak, 0),
			COALESCE(p.recent_pain, 0)
		FROM words w
		JOIN word_topics wt ON wt.word_id = w.id
		JOIN topics t ON t.id = wt.topic_id
		LEFT JOIN (
			SELECT word_id,
				SUM(seen_count) AS seen_count,
				SUM(correct_count) AS correct_count,
				SUM(wrong_count) AS wrong_count,
				MAX(correct_streak) AS correct_streak,
				MAX(recent_pain) AS recent_pain
			FROM training_progress
			GROUP BY word_id
		) p ON p.word_id = w.id
		GROUP BY w.id, w.spanish, w.russian, p.seen_count, p.correct_count, p.wrong_count, p.correct_streak, p.recent_pain
		ORDER BY COALESCE(p.recent_pain, 0) DESC, COALESCE(p.seen_count, 0) ASC, GROUP_CONCAT(t.name, ', '), w.spanish_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list training stats: %w", err)
	}
	defer rows.Close()

	stats := make([]domain.TrainingWordStats, 0)
	for rows.Next() {
		var item domain.TrainingWordStats
		if err := rows.Scan(
			&item.WordID,
			&item.TopicName,
			&item.Spanish,
			&item.Russian,
			&item.SeenCount,
			&item.CorrectCount,
			&item.WrongCount,
			&item.CorrectStreak,
			&item.RecentPain,
		); err != nil {
			return nil, fmt.Errorf("scan training stats: %w", err)
		}
		if item.SeenCount > 0 {
			item.Accuracy = item.CorrectCount * 100 / item.SeenCount
		}
		stats = append(stats, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training stats: %w", err)
	}
	return stats, nil
}
