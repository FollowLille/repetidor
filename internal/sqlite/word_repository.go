package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"repetidor/internal/domain"
)

type WordRepository struct {
	db *sql.DB
}

func NewWordRepository(db *sql.DB) *WordRepository {
	return &WordRepository{db: db}
}

func (r *WordRepository) ListByTopicID(ctx context.Context, topicID int64) ([]domain.Word, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, topic_id, spanish, russian, notes, created_at, updated_at
		FROM words
		WHERE topic_id = ?
		ORDER BY spanish ASC, russian ASC
	`, topicID)
	if err != nil {
		return nil, fmt.Errorf("list words by topic id: %w", err)
	}
	defer rows.Close()

	words := make([]domain.Word, 0)
	for rows.Next() {
		var word domain.Word
		if err := rows.Scan(&word.ID, &word.TopicID, &word.Spanish, &word.Russian, &word.Notes, &word.CreatedAt, &word.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan word: %w", err)
		}
		words = append(words, word)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate words by topic id: %w", err)
	}

	return words, nil
}

func (r *WordRepository) Create(ctx context.Context, word domain.Word) (domain.Word, error) {
	created := domain.Word{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO words(topic_id, spanish, russian, notes)
		VALUES(?, ?, ?, ?)
		RETURNING id, topic_id, spanish, russian, notes, created_at, updated_at
	`, word.TopicID, word.Spanish, word.Russian, word.Notes).Scan(
		&created.ID,
		&created.TopicID,
		&created.Spanish,
		&created.Russian,
		&created.Notes,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Word{}, fmt.Errorf("create word: %w", err)
	}
	return created, nil
}
