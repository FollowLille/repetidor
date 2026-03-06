package handlers

import "net/http"

type Container struct {
	Home     http.Handler
	Training http.Handler
	Topic    http.Handler
}

func NewContainer() (*Container, error) {
	homeHandler, err := NewHomeHandler()
	if err != nil {
		return nil, err
	}

	trainingHandler, err := NewTrainingHandler()
	if err != nil {
		return nil, err
	}

	topicHandler, err := NewTopicHandler()
	if err != nil {
		return nil, err
	}

	return &Container{
		Home:     homeHandler,
		Training: trainingHandler,
		Topic:    topicHandler,
	}, nil
}
