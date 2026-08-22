package domain

import "time"

type LearningCourse struct {
	ID              int64
	LanguageTrackID int64
	ParentID        *int64
	Name            string
	Description     string
	SortOrder       int
	TopicIDs        []int64
	PrerequisiteIDs []int64
	LevelIDs        []int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type LearningLevel struct {
	ID, TrackID             int64
	Code, Name, Description string
	SortOrder               int
}
