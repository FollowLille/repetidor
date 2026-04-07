package domain

import "time"

// Topic represents a vocabulary topic with notes or description.
type Topic struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
