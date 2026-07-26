package handlers

import (
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type Container struct {
	Home      *HomeHandler
	Training  *TrainingHandler
	Topics    *TopicsHandler
	Topic     *TopicHandler
	TopicEdit *TopicEditHandler
	WordEdit  *WordEditHandler
	Stats     *StatsHandler
}

func NewContainer(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, trainingRepo storage.TrainingRepository, appLogger logger.Logger) (*Container, error) {
	homeHandler, err := NewHomeHandler(topicRepo)
	if err != nil {
		return nil, err
	}

	trainingHandler, err := NewTrainingHandler(topicRepo, wordRepo, trainingRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicsHandler, err := NewTopicsHandler(topicRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicHandler, err := NewTopicHandler(topicRepo, wordRepo, appLogger)
	if err != nil {
		return nil, err
	}

	topicEditHandler, err := NewTopicEditHandler(topicRepo, appLogger)
	if err != nil {
		return nil, err
	}
	wordEditHandler, err := NewWordEditHandler(topicRepo, wordRepo, appLogger)
	if err != nil {
		return nil, err
	}
	statsHandler, err := NewStatsHandler(trainingRepo, appLogger)
	if err != nil {
		return nil, err
	}

	return &Container{Home: homeHandler, Training: trainingHandler, Topics: topicsHandler, Topic: topicHandler, TopicEdit: topicEditHandler, WordEdit: wordEditHandler, Stats: statsHandler}, nil
}
