package domain

import "time"

const (
	SessionActive    = "active"
	SessionCompleted = "completed"
	SessionAbandoned = "abandoned"
)

type TrainingSession struct {
	ID            int64
	Mode          string
	DirectionMode string
	AnswerMode    string
	TopicID       int64
	Size          int
	Completed     int
	Correct       int
	Skipped       int
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
	AbandonedAt   *time.Time
}

type TrainingSessionCard struct {
	SessionID    int64
	Position     int
	WordID       int64
	TopicID      int64
	Direction    string
	AnswerMode   string
	Answered     bool
	Correct      bool
	Response     string
	ErrorKind    string
	EditDistance int
	AnsweredAt   *time.Time
	Word         Word
	Topic        Topic
}

type FrequentMistake struct {
	WordID           int64
	Spanish, Russian string
	Count            int
	Typos            int
	Skips            int
}
