package domain

import "time"

type Topic struct {
	ID          int64
	CourseID    int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
