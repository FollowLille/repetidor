package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type TopicRepository struct {
	db *sql.DB
}

func NewTopicRepository(db *sql.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (r *TopicRepository) List(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, language_track_id, name, description, created_at, updated_at
		FROM topics
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	topics := make([]domain.Topic, 0)
	for rows.Next() {
		var topic domain.Topic
		if err := rows.Scan(&topic.ID, &topic.CourseID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, topic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topics: %w", err)
	}

	return topics, nil
}

func (r *TopicRepository) ListByCourse(ctx context.Context, courseID int64) ([]domain.Topic, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, language_track_id, name, description, created_at, updated_at FROM topics WHERE language_track_id=? ORDER BY name ASC`, courseID)
	if err != nil {
		return nil, fmt.Errorf("list topics by course: %w", err)
	}
	defer rows.Close()
	var topics []domain.Topic
	for rows.Next() {
		var topic domain.Topic
		if err := rows.Scan(&topic.ID, &topic.CourseID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

func (r *TopicRepository) Create(ctx context.Context, topic domain.Topic) (domain.Topic, error) {
	created := domain.Topic{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO topics(language_track_id, name, description)
		VALUES(?, ?, ?)
		RETURNING id, language_track_id, name, description, created_at, updated_at
	`, topic.CourseID, topic.Name, topic.Description).Scan(
		&created.ID,
		&created.CourseID,
		&created.Name,
		&created.Description,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.Topic{}, fmt.Errorf("create topic: %w", storage.ErrTopicAlreadyExists)
		}
		return domain.Topic{}, fmt.Errorf("create topic: %w", err)
	}
	return created, nil
}

func (r *TopicRepository) GetByName(ctx context.Context, name string) (domain.Topic, error) {
	var topic domain.Topic
	err := r.db.QueryRowContext(ctx, `
		SELECT id, language_track_id, name, description, created_at, updated_at
		FROM topics
		WHERE name = ?
	`, name).Scan(&topic.ID, &topic.CourseID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Topic{}, storage.ErrTopicNotFound
	}
	if err != nil {
		return domain.Topic{}, fmt.Errorf("get topic by name: %w", err)
	}
	return topic, nil
}

func (r *TopicRepository) Update(ctx context.Context, topic domain.Topic) (domain.Topic, error) {
	updated := domain.Topic{}
	err := r.db.QueryRowContext(ctx, `
		UPDATE topics
		SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		RETURNING id, language_track_id, name, description, created_at, updated_at
	`, topic.Name, topic.Description, topic.ID).Scan(
		&updated.ID,
		&updated.CourseID,
		&updated.Name,
		&updated.Description,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Topic{}, storage.ErrTopicNotFound
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.Topic{}, fmt.Errorf("update topic: %w", storage.ErrTopicAlreadyExists)
		}
		return domain.Topic{}, fmt.Errorf("update topic: %w", err)
	}
	return updated, nil
}

func (r *TopicRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin topic deletion: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE words
		SET topic_id = (
			SELECT MIN(wt.topic_id)
			FROM word_topics wt
			WHERE wt.word_id = words.id AND wt.topic_id <> ?
		)
		WHERE topic_id = ?
		  AND EXISTS (
			SELECT 1 FROM word_topics wt
			WHERE wt.word_id = words.id AND wt.topic_id <> ?
		  )
	`, id, id, id)
	if err != nil {
		return fmt.Errorf("move shared words before topic deletion: %w", err)
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM topics WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete topic: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted topics: %w", err)
	}
	if deleted == 0 {
		return storage.ErrTopicNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit topic deletion: %w", err)
	}
	return nil
}
