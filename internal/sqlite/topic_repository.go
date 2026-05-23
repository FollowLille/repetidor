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
	rows, err := r.db.QueryContext(ctx, "SELECT name, notes FROM topics ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	topics := make([]domain.Topic, 0)
	for rows.Next() {
		var topic domain.Topic
		if err := rows.Scan(&topic.Name, &topic.Notes); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, topic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topics: %w", err)
	}

	return topics, nil
}

func (r *TopicRepository) Create(ctx context.Context, topic domain.Topic) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO topics(name, notes) VALUES(?, ?)", topic.Name, topic.Notes)
	if err != nil {
		return fmt.Errorf("create topic: %w", err)
	}
	return nil
}

func (r *TopicRepository) GetByName(ctx context.Context, name string) (domain.Topic, error) {
	var topic domain.Topic
	err := r.db.QueryRowContext(ctx, "SELECT name, notes FROM topics WHERE name = ?", name).Scan(&topic.Name, &topic.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Topic{}, storage.ErrTopicNotFound
	}
	if err != nil {
		return domain.Topic{}, fmt.Errorf("get topic by name: %w", err)
	}
	return topic, nil
}
