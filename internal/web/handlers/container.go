package handlers

import (
	"net/http"

	"repetidor/internal/storage"
)

// Container groups HTTP handlers used by the web layer.
type Container struct {
	Home     http.Handler
	Topics   http.Handler
	Training http.Handler
	Topic    http.Handler
}

// NewContainer creates and wires web handlers.
func NewContainer(topicRepository storage.TopicRepository) (*Container, error) {
	homeHandler, err := NewHomeHandler(topicRepository)
	if err != nil {
		return nil, err
	}

	topicsHandler, err := NewTopicsHandler(topicRepository)
	if err != nil {
		return nil, err
	}

	trainingHandler, err := NewTrainingHandler()
	if err != nil {
		return nil, err
	}

	topicHandler, err := NewTopicHandler(topicRepository)
	if err != nil {
		return nil, err
	}

	return &Container{
		Home:     homeHandler,
		Topics:   topicsHandler,
		Training: trainingHandler,
		Topic:    topicHandler,
	}, nil
}
