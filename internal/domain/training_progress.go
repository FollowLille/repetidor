package domain

import "time"

type TrainingProgress struct {
	WordID        int64
	Direction     string
	SeenCount     int
	CorrectCount  int
	WrongCount    int
	CorrectStreak int
	RecentPain    int
	LastSeenAt    *time.Time
}

type TrainingWordStats struct {
	WordID        int64
	TopicName     string
	Spanish       string
	Russian       string
	SeenCount     int
	CorrectCount  int
	WrongCount    int
	CorrectStreak int
	RecentPain    int
	Accuracy      int
}
