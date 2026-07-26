package storage

import (
	"context"
	"errors"
	"repetidor/internal/domain"
)

var (
	ErrWordNotFound      = errors.New("word not found")
	ErrWordAlreadyExists = errors.New("word already exists")
)

type WordRepository interface {
	ListByTopicID(ctx context.Context, topicID int64) ([]domain.Word, error)
	GetByID(ctx context.Context, topicID, wordID int64) (domain.Word, error)
	Create(ctx context.Context, word domain.Word) (domain.Word, error)
	Update(ctx context.Context, word domain.Word) (domain.Word, error)
	Delete(ctx context.Context, topicID, wordID int64) error
}
