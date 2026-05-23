package storage

import (
	"context"
	"errors"
	"repetidor/internal/domain"
)

var ErrWordNotFound = errors.New("word not found")

type WordRepository interface {
	ListByTopicID(ctx context.Context, topicID int64) ([]domain.Word, error)
	Create(ctx context.Context, word domain.Word) (domain.Word, error)
}
