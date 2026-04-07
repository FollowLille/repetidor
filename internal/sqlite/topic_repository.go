package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"repetidor/internal/domain"
)

type TopicRepository struct {
	db *sql.DB
}

func NewTopicRepository(db *sql.DB) *TopicRepository {
	return &TopicRepository{
		db: db,
	}
}

// Create inserts a new topic and returns the stored entity.
func (r *TopicRepository) Create(ctx context.Context, topic domain.Topic) (domain.Topic, error) {
	query := `
		INSERT INTO topics (name, description) VALUES (?, ?)
		RETURNING id, name, description, created_at, updated_at;
    `

	var created domain.Topic

	err := r.db.QueryRowContext(
		ctx, query, topic.Name, topic.Description,
	).Scan(
		&created.ID, &created.Name, &created.Description, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("create topic: %w", err)
	}

	return created, nil
}

// List returns all topics ordered by name.
func (r *TopicRepository) List(ctx context.Context) ([]domain.Topic, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM topics
		ORDER BY name ASC;
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	topics := make([]domain.Topic, 0)

	for rows.Next() {
		var topic domain.Topic

		if err := rows.Scan(
			&topic.ID,
			&topic.Name,
			&topic.Description,
			&topic.CreatedAt,
			&topic.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan topic row: %w", err)
		}

		topics = append(topics, topic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic rows: %w", err)
	}

	return topics, nil
}

// GetByName returns a topic by its name.
func (r *TopicRepository) GetByName(ctx context.Context, name string) (domain.Topic, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM topics
		WHERE name = ?
		LIMIT 1;
	`

	var topic domain.Topic

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&topic.ID,
		&topic.Name,
		&topic.Description,
		&topic.CreatedAt,
		&topic.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Topic{}, fmt.Errorf("get topic by name %q: %w", name, sql.ErrNoRows)
		}

		return domain.Topic{}, fmt.Errorf("get topic by name %q: %w", name, err)
	}

	return topic, nil
}
