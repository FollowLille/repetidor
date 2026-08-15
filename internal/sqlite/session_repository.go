package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type SessionRepository struct{ db *sql.DB }

func NewSessionRepository(db *sql.DB) *SessionRepository { return &SessionRepository{db: db} }

func (r *SessionRepository) Create(ctx context.Context, session domain.TrainingSession, cards []domain.TrainingSessionCard) (domain.TrainingSession, error) {
	if len(cards) == 0 {
		return domain.TrainingSession{}, fmt.Errorf("create session: empty card queue")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.TrainingSession{}, err
	}
	defer tx.Rollback()
	var topicID any
	if session.TopicID > 0 {
		topicID = session.TopicID
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO training_sessions(mode, topic_id, size, direction_mode, answer_mode) VALUES(?, ?, ?, ?, ?)`, session.Mode, topicID, len(cards), defaultString(session.DirectionMode, "auto"), defaultString(session.AnswerMode, "auto"))
	if err != nil {
		return domain.TrainingSession{}, fmt.Errorf("insert session: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.TrainingSession{}, err
	}
	for i, card := range cards {
		if _, err := tx.ExecContext(ctx, `INSERT INTO training_session_cards(session_id, position, word_id, topic_id, direction, answer_mode) VALUES(?, ?, ?, ?, ?, ?)`, id, i+1, card.WordID, card.TopicID, card.Direction, card.AnswerMode); err != nil {
			return domain.TrainingSession{}, fmt.Errorf("insert session card: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.TrainingSession{}, err
	}
	return r.Get(ctx, id)
}

func (r *SessionRepository) Get(ctx context.Context, id int64) (domain.TrainingSession, error) {
	row := r.db.QueryRowContext(ctx, sessionSelect+` WHERE id = ?`, id)
	return scanSession(row)
}

func (r *SessionRepository) CurrentCard(ctx context.Context, sessionID int64) (domain.TrainingSessionCard, error) {
	var position int
	err := r.db.QueryRowContext(ctx, `SELECT completed + 1 FROM training_sessions WHERE id = ? AND status = 'active'`, sessionID).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TrainingSessionCard{}, storage.ErrSessionComplete
	}
	if err != nil {
		return domain.TrainingSessionCard{}, err
	}
	return r.GetCard(ctx, sessionID, position)
}

func (r *SessionRepository) GetCard(ctx context.Context, sessionID int64, position int) (domain.TrainingSessionCard, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT c.session_id, c.position, c.word_id, c.topic_id, c.direction, c.answer_mode,
		       c.answered, c.is_correct, c.response, c.error_kind, c.edit_distance, c.answered_at,
		       w.spanish, w.russian, w.notes, t.name, t.description
		FROM training_session_cards c
		JOIN words w ON w.id = c.word_id
		JOIN topics t ON t.id = c.topic_id
		WHERE c.session_id = ? AND c.position = ?`, sessionID, position)
	var card domain.TrainingSessionCard
	var answered, correct int
	var answeredAt sql.NullTime
	err := row.Scan(&card.SessionID, &card.Position, &card.WordID, &card.TopicID, &card.Direction, &card.AnswerMode, &answered, &correct, &card.Response, &card.ErrorKind, &card.EditDistance, &answeredAt, &card.Word.Spanish, &card.Word.Russian, &card.Word.Notes, &card.Topic.Name, &card.Topic.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return card, storage.ErrSessionCardNotFound
	}
	if err != nil {
		return card, err
	}
	card.Answered, card.Correct = answered != 0, correct != 0
	card.Word.ID, card.Word.TopicID = card.WordID, card.TopicID
	card.Topic.ID = card.TopicID
	if answeredAt.Valid {
		card.AnsweredAt = &answeredAt.Time
	}
	return card, nil
}

func (r *SessionRepository) ListCards(ctx context.Context, sessionID int64) ([]domain.TrainingSessionCard, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.session_id, c.position, c.word_id, c.topic_id, c.direction, c.answer_mode,
		       c.answered, c.is_correct, c.response, c.error_kind, c.edit_distance, c.answered_at,
		       w.spanish, w.russian, w.notes, t.name, t.description
		FROM training_session_cards c
		JOIN words w ON w.id = c.word_id
		JOIN topics t ON t.id = c.topic_id
		WHERE c.session_id = ? ORDER BY c.position`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := make([]domain.TrainingSessionCard, 0)
	for rows.Next() {
		card, err := scanSessionCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func (r *SessionRepository) RequeueCard(ctx context.Context, sessionID int64, position int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO training_session_cards(session_id, position, word_id, topic_id, direction, answer_mode)
		SELECT session_id, (SELECT MAX(position)+1 FROM training_session_cards WHERE session_id = ?), word_id, topic_id, direction, answer_mode
		FROM training_session_cards source
		WHERE source.session_id = ? AND source.position = ? AND source.answered = 1 AND source.is_correct = 0
		AND EXISTS (SELECT 1 FROM training_sessions s WHERE s.id=source.session_id AND s.abandoned_at IS NULL)
		AND NOT EXISTS (SELECT 1 FROM training_session_cards pending WHERE pending.session_id=source.session_id AND pending.answered=0 AND pending.word_id=source.word_id AND pending.direction=source.direction)`, sessionID, sessionID, position)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return storage.ErrSessionCardNotFound
	}
	result, err = tx.ExecContext(ctx, `UPDATE training_sessions SET size=size+1, status='active', completed_at=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=? AND abandoned_at IS NULL`, sessionID)
	if err != nil {
		return err
	}
	affected, _ = result.RowsAffected()
	if affected != 1 {
		return storage.ErrSessionComplete
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *SessionRepository) Abandon(ctx context.Context, sessionID int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE training_sessions SET status = 'completed', abandoned_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 'active'`, sessionID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return storage.ErrSessionComplete
	}
	return nil
}

func (r *SessionRepository) ListRecent(ctx context.Context, limit int) ([]domain.TrainingSession, error) {
	if limit < 1 || limit > 100 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, sessionSelect+` ORDER BY updated_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.TrainingSession, 0)
	for rows.Next() {
		item, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SessionRepository) MistakeWordIDs(ctx context.Context, sessionID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT word_id FROM training_session_cards WHERE session_id = ? AND answered = 1 AND is_correct = 0 AND error_kind <> 'skipped' ORDER BY position`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SessionRepository) ListFrequentMistakes(ctx context.Context, limit int) ([]domain.FrequentMistake, error) {
	if limit < 1 || limit > 100 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `SELECT c.word_id, w.spanish, w.russian, SUM(CASE WHEN c.error_kind<>'skipped' THEN 1 ELSE 0 END), SUM(CASE WHEN c.error_kind='typo' THEN 1 ELSE 0 END), SUM(CASE WHEN c.error_kind='skipped' THEN 1 ELSE 0 END)
		FROM training_session_cards c JOIN words w ON w.id=c.word_id
		WHERE c.answered=1 AND c.is_correct=0 GROUP BY c.word_id, w.spanish, w.russian ORDER BY SUM(CASE WHEN c.error_kind<>'skipped' THEN 1 ELSE 0 END) DESC, c.word_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FrequentMistake
	for rows.Next() {
		var item domain.FrequentMistake
		if err := rows.Scan(&item.WordID, &item.Spanish, &item.Russian, &item.Count, &item.Typos, &item.Skips); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

const sessionSelect = `SELECT id, mode, direction_mode, answer_mode, COALESCE(topic_id, 0), size, completed, correct, skipped,
	CASE WHEN abandoned_at IS NOT NULL THEN 'abandoned' ELSE status END,
	created_at, updated_at, completed_at, abandoned_at FROM training_sessions`

func scanSession(row rowScanner) (domain.TrainingSession, error) {
	var item domain.TrainingSession
	var completedAt sql.NullTime
	var abandonedAt sql.NullTime
	err := row.Scan(&item.ID, &item.Mode, &item.DirectionMode, &item.AnswerMode, &item.TopicID, &item.Size, &item.Completed, &item.Correct, &item.Skipped, &item.Status, &item.CreatedAt, &item.UpdatedAt, &completedAt, &abandonedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return item, storage.ErrSessionNotFound
	}
	if err != nil {
		return item, err
	}
	if completedAt.Valid {
		value := completedAt.Time
		item.CompletedAt = &value
	}
	if abandonedAt.Valid {
		value := abandonedAt.Time
		item.AbandonedAt = &value
	}
	return item, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func scanSessionCard(row rowScanner) (domain.TrainingSessionCard, error) {
	var card domain.TrainingSessionCard
	var answered, correct int
	var answeredAt sql.NullTime
	err := row.Scan(&card.SessionID, &card.Position, &card.WordID, &card.TopicID, &card.Direction, &card.AnswerMode, &answered, &correct, &card.Response, &card.ErrorKind, &card.EditDistance, &answeredAt, &card.Word.Spanish, &card.Word.Russian, &card.Word.Notes, &card.Topic.Name, &card.Topic.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return card, storage.ErrSessionCardNotFound
	}
	if err != nil {
		return card, err
	}
	card.Answered, card.Correct = answered != 0, correct != 0
	card.Word.ID, card.Word.TopicID, card.Topic.ID = card.WordID, card.TopicID, card.TopicID
	if answeredAt.Valid {
		card.AnsweredAt = &answeredAt.Time
	}
	return card, nil
}

var _ storage.SessionRepository = (*SessionRepository)(nil)
