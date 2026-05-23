package storage

import (
	"context"
	"errors"
	"repetidor/internal/domain"
)

var ErrTopicNotFound = errors.New("topic not found")

type TopicRepository interface {
	List(ctx context.Context) ([]domain.Topic, error)
	Create(ctx context.Context, topic domain.Topic) (domain.Topic, error)
	GetByName(ctx context.Context, name string) (domain.Topic, error)
}
