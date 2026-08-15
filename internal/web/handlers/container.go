package handlers

import (
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type Container struct {
	Home           *HomeHandler
	Training       *TrainingHandler
	Topics         *TopicsHandler
	Topic          *TopicHandler
	TopicEdit      *TopicEditHandler
	WordEdit       *WordEditHandler
	Stats          *StatsHandler
	Session        *SessionHandler
	Arena          *ArenaHandler
	Settings       *SettingsHandler
	Import         *ImportHandler
	Courses        *CoursesHandler
	Course         *CourseHandler
	Boards         *BoardsHandler
	CoursePackages *CoursePackageHandler
}

func NewContainer(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, trainingRepo storage.TrainingRepository, sessionRepo storage.SessionRepository, courseRepo storage.CourseRepository, learningCourseRepo storage.LearningCourseRepository, theoryRepo storage.TheoryRepository, boardRepo storage.BoardRepository, packageRepo storage.CoursePackageRepository, appLogger logger.Logger) (*Container, error) {
	homeHandler, err := NewHomeHandler(topicRepo, sessionRepo, courseRepo)
	if err != nil {
		return nil, err
	}

	trainingHandler, err := NewTrainingHandler(topicRepo, wordRepo, trainingRepo, sessionRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicsHandler, err := NewTopicsHandler(topicRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicHandler, err := NewTopicHandler(topicRepo, wordRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicEditHandler, err := NewTopicEditHandler(topicRepo, appLogger)
	if err != nil {
		return nil, err
	}
	wordEditHandler, err := NewWordEditHandler(topicRepo, wordRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	statsHandler, err := NewStatsHandler(trainingRepo, sessionRepo, appLogger)
	if err != nil {
		return nil, err
	}
	sessionHandler, err := NewSessionHandler(sessionRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	arenaHandler, err := NewArenaHandler(topicRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	settingsHandler, err := NewSettingsHandler(courseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	importHandler, err := NewImportHandler(topicRepo, wordRepo, courseRepo, learningCourseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	coursesHandler, err := NewCoursesHandler(learningCourseRepo, topicRepo, courseRepo, theoryRepo, appLogger)
	if err != nil {
		return nil, err
	}
	courseHandler, err := NewCourseHandler(learningCourseRepo, theoryRepo, courseRepo, topicRepo, boardRepo, appLogger)
	if err != nil {
		return nil, err
	}
	boardsHandler, err := NewBoardsHandler(boardRepo, learningCourseRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}
	packageHandler, err := NewCoursePackageHandler(packageRepo, courseRepo, appLogger)
	if err != nil {
		return nil, err
	}

	return &Container{Home: homeHandler, Training: trainingHandler, Topics: topicsHandler, Topic: topicHandler, TopicEdit: topicEditHandler, WordEdit: wordEditHandler, Stats: statsHandler, Session: sessionHandler, Arena: arenaHandler, Settings: settingsHandler, Import: importHandler, Courses: coursesHandler, Course: courseHandler, Boards: boardsHandler, CoursePackages: packageHandler}, nil
}
