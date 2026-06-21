package domain

import "time"

type Word struct {
	ID         int64
	TopicID    int64
	Spanish    string
	SpanishKey string
	Russian    string
	RussianKey string
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
