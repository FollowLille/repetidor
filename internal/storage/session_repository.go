package storage

import (
	"context"
	"errors"

	"repetidor/internal/domain"
)

var (
	ErrSessionNotFound     = errors.New("training session not found")
	ErrSessionCardNotFound = errors.New("training session card not found")
	ErrSessionComplete     = errors.New("training session is complete")
)

type SessionRepository interface {
	Create(ctx context.Context, session domain.TrainingSession, cards []domain.TrainingSessionCard) (domain.TrainingSession, error)
	Get(ctx context.Context, id int64) (domain.TrainingSession, error)
	CurrentCard(ctx context.Context, sessionID int64) (domain.TrainingSessionCard, error)
	GetCard(ctx context.Context, sessionID int64, position int) (domain.TrainingSessionCard, error)
	ListCards(ctx context.Context, sessionID int64) ([]domain.TrainingSessionCard, error)
	RequeueCard(ctx context.Context, sessionID int64, position int) error
	Abandon(ctx context.Context, sessionID int64) error
	ListRecent(ctx context.Context, limit int) ([]domain.TrainingSession, error)
	MistakeWordIDs(ctx context.Context, sessionID int64) ([]int64, error)
	ListFrequentMistakes(ctx context.Context, limit int) ([]domain.FrequentMistake, error)
}
