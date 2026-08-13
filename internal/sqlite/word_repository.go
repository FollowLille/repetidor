package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"repetidor/internal/domain"
	"repetidor/internal/storage"
	"strings"
)

type WordRepository struct {
	db *sql.DB
}

func NewWordRepository(db *sql.DB) *WordRepository {
	return &WordRepository{db: db}
}

func normalizeWordKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

func (r *WordRepository) ListByTopicID(ctx context.Context, topicID int64) ([]domain.Word, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT w.id, ?, w.spanish, w.spanish_key, w.russian, w.russian_key, w.notes, w.created_at, w.updated_at
		FROM words w
		JOIN word_topics wt ON wt.word_id = w.id
		WHERE wt.topic_id = ?
		ORDER BY w.spanish_key ASC, w.russian_key ASC
	`, topicID, topicID)
	if err != nil {
		return nil, fmt.Errorf("list words by topic id: %w", err)
	}
	defer rows.Close()

	words := make([]domain.Word, 0)
	for rows.Next() {
		var word domain.Word
		if err := rows.Scan(&word.ID, &word.TopicID, &word.Spanish, &word.SpanishKey, &word.Russian, &word.RussianKey, &word.Notes, &word.CreatedAt, &word.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan word: %w", err)
		}
		words = append(words, word)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate words by topic id: %w", err)
	}

	return words, nil
}

func (r *WordRepository) GetByID(ctx context.Context, topicID, wordID int64) (domain.Word, error) {
	var word domain.Word
	err := r.db.QueryRowContext(ctx, `
		SELECT w.id, ?, w.spanish, w.spanish_key, w.russian, w.russian_key, w.notes, w.created_at, w.updated_at
		FROM words w
		JOIN word_topics wt ON wt.word_id = w.id
		WHERE w.id = ? AND wt.topic_id = ?
	`, topicID, wordID, topicID).Scan(
		&word.ID,
		&word.TopicID,
		&word.Spanish,
		&word.SpanishKey,
		&word.Russian,
		&word.RussianKey,
		&word.Notes,
		&word.CreatedAt,
		&word.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return domain.Word{}, storage.ErrWordNotFound
	}
	if err != nil {
		return domain.Word{}, fmt.Errorf("get word by id: %w", err)
	}
	return word, nil
}

func (r *WordRepository) Create(ctx context.Context, word domain.Word) (domain.Word, error) {
	word.SpanishKey = normalizeWordKey(word.Spanish)
	word.RussianKey = normalizeWordKey(word.Russian)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Word{}, fmt.Errorf("begin word creation: %w", err)
	}
	defer tx.Rollback()

	created := domain.Word{TopicID: word.TopicID}
	err = tx.QueryRowContext(ctx, `
		SELECT id, spanish, spanish_key, russian, russian_key, notes, created_at, updated_at
		FROM words
		WHERE spanish_key = ? AND russian_key = ?
		ORDER BY id
		LIMIT 1
	`, word.SpanishKey, word.RussianKey).Scan(
		&created.ID,
		&created.Spanish,
		&created.SpanishKey,
		&created.Russian,
		&created.RussianKey,
		&created.Notes,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
		INSERT INTO words(topic_id, spanish, spanish_key, russian, russian_key, notes)
		VALUES(?, ?, ?, ?, ?, ?)
		RETURNING id, spanish, spanish_key, russian, russian_key, notes, created_at, updated_at
	`, word.TopicID, word.Spanish, word.SpanishKey, word.Russian, word.RussianKey, word.Notes).Scan(
			&created.ID,
			&created.Spanish,
			&created.SpanishKey,
			&created.Russian,
			&created.RussianKey,
			&created.Notes,
			&created.CreatedAt,
			&created.UpdatedAt,
		)
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.Word{}, fmt.Errorf("create word: %w", storage.ErrWordAlreadyExists)
		}
		return domain.Word{}, fmt.Errorf("create word: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO word_topics(word_id, topic_id) VALUES(?, ?)`, created.ID, word.TopicID); err != nil {
		if isUniqueConstraintError(err) {
			return domain.Word{}, fmt.Errorf("link word to topic: %w", storage.ErrWordAlreadyExists)
		}
		return domain.Word{}, fmt.Errorf("link word to topic: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Word{}, fmt.Errorf("commit word creation: %w", err)
	}
	return created, nil
}

func (r *WordRepository) Update(ctx context.Context, word domain.Word) (domain.Word, error) {
	word.SpanishKey = normalizeWordKey(word.Spanish)
	word.RussianKey = normalizeWordKey(word.Russian)

	var updated domain.Word
	err := r.db.QueryRowContext(ctx, `
		UPDATE words
		SET spanish = ?, spanish_key = ?, russian = ?, russian_key = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND EXISTS (SELECT 1 FROM word_topics WHERE word_id = words.id AND topic_id = ?)
		RETURNING id, spanish, spanish_key, russian, russian_key, notes, created_at, updated_at
	`, word.Spanish, word.SpanishKey, word.Russian, word.RussianKey, word.Notes, word.ID, word.TopicID).Scan(
		&updated.ID,
		&updated.Spanish,
		&updated.SpanishKey,
		&updated.Russian,
		&updated.RussianKey,
		&updated.Notes,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	updated.TopicID = word.TopicID
	if err == sql.ErrNoRows {
		return domain.Word{}, storage.ErrWordNotFound
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.Word{}, fmt.Errorf("update word: %w", storage.ErrWordAlreadyExists)
		}
		return domain.Word{}, fmt.Errorf("update word: %w", err)
	}
	return updated, nil
}

func (r *WordRepository) Delete(ctx context.Context, topicID, wordID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin word unlink: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `DELETE FROM word_topics WHERE word_id = ? AND topic_id = ?`, wordID, topicID)
	if err != nil {
		return fmt.Errorf("unlink word from topic: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count unlinked words: %w", err)
	}
	if deleted == 0 {
		return storage.ErrWordNotFound
	}

	var links int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM word_topics WHERE word_id = ?`, wordID).Scan(&links); err != nil {
		return fmt.Errorf("count word topics: %w", err)
	}
	if links == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM words WHERE id = ?`, wordID); err != nil {
			return fmt.Errorf("delete unlinked word: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit word unlink: %w", err)
	}
	return nil
}
