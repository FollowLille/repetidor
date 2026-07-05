package domain

type TrainingProgress struct {
	WordID        int64
	Direction     string
	SeenCount     int
	CorrectCount  int
	WrongCount    int
	CorrectStreak int
	RecentPain    int
}
