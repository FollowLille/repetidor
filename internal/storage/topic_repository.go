package storage

import (
	"context"

	"repetidor/internal/domain"
)

// TopicRepository defines storage operations for topics.
type TopicRepository interface {
	Create(ctx context.Context, topic domain.Topic) (domain.Topic, error)
	List(ctx context.Context) ([]domain.Topic, error)
	GetByName(ctx context.Context, name string) (domain.Topic, error)
}
