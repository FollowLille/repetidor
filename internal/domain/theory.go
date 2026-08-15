package domain

import "time"

type TheoryBlock struct {
	ID        int64
	CourseID  int64
	Kind      string
	Title     string
	Content   string
	SortOrder int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TheoryExercise struct {
	ID            int64
	CourseID      int64
	TheoryBlockID *int64
	Kind          string
	Prompt        string
	Options       []string
	CorrectAnswer string
	Explanation   string
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CourseProgress struct {
	CourseID            int64
	TheoryReadAt        *time.Time
	PracticeCompletedAt *time.Time
	PracticeCorrect     int
	PracticeTotal       int
	ExerciseCount       int
	UpdatedAt           time.Time
}

func (p CourseProgress) Percent() int {
	parts, completed := 1, 0
	if p.TheoryReadAt != nil {
		completed++
	}
	if p.ExerciseCount > 0 {
		parts++
		if p.PracticeCompletedAt != nil {
			completed++
		}
	}
	return completed * 100 / parts
}

type TheoryAnswerResult struct {
	Correct     bool
	Expected    string
	Explanation string
	Progress    CourseProgress
}
