package storage

import (
	"context"
	"errors"
	"repetidor/internal/domain"
)

var ErrWordNotFound = errors.New("word not found")

type WordRepository interface {
	List(ctx context.Context) ([]domain.Word, error)
	ListByTopicID(ctx context.Context, topicID int64) ([]domain.Word, error)
	GetByID(ctx context.Context, id int64) (domain.Word, error)
	Create(ctx context.Context, word domain.Word) (domain.Word, error)
}
