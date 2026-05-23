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
		SELECT id, name, description, created_at, updated_at
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
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, topic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topics: %w", err)
	}

	return topics, nil
}

func (r *TopicRepository) Create(ctx context.Context, topic domain.Topic) (domain.Topic, error) {
	created := domain.Topic{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO topics(name, description)
		VALUES(?, ?)
		RETURNING id, name, description, created_at, updated_at
	`, topic.Name, topic.Description).Scan(
		&created.ID,
		&created.Name,
		&created.Description,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("create topic: %w", err)
	}
	return created, nil
}

func (r *TopicRepository) GetByName(ctx context.Context, name string) (domain.Topic, error) {
	var topic domain.Topic
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM topics
		WHERE name = ?
	`, name).Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Topic{}, storage.ErrTopicNotFound
	}
	if err != nil {
		return domain.Topic{}, fmt.Errorf("get topic by name: %w", err)
	}
	return topic, nil
}
