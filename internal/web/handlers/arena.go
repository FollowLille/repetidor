package handlers

import (
	"html/template"
	"net/http"

	"repetidor/internal/game"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type ArenaHandler struct {
	templates  *template.Template
	topicRepo  storage.TopicRepository
	courseRepo storage.CourseRepository
	logger     logger.Logger
}

func NewArenaHandler(topicRepo storage.TopicRepository, courseRepo storage.CourseRepository, appLogger logger.Logger) (*ArenaHandler, error) {
	tmpl, err := parsePage("arena.html")
	if err != nil {
		return nil, err
	}
	return &ArenaHandler{templates: tmpl, topicRepo: topicRepo, courseRepo: courseRepo, logger: appLogger}, nil
}

func (h *ArenaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	course := activeCourse(h.courseRepo, r)
	topics, err := h.topicRepo.ListByCourse(r.Context(), course.ID)
	if err != nil {
		h.logger.Error("failed to load arena topics", "error", err)
		http.Error(w, "failed to load arena", http.StatusInternalServerError)
		return
	}
	data := pageData(r, map[string]any{"Title": "Practice arena", "Modes": game.Modes, "Topics": topics, "Course": course})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render arena", "error", err)
	}
}
