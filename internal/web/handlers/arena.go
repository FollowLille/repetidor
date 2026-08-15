package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"repetidor/internal/game"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type ArenaHandler struct {
	templates *template.Template
	topicRepo storage.TopicRepository
	logger    logger.Logger
}

func NewArenaHandler(topicRepo storage.TopicRepository, appLogger logger.Logger) (*ArenaHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "arena.html"))
	if err != nil {
		return nil, err
	}
	return &ArenaHandler{templates: tmpl, topicRepo: topicRepo, logger: appLogger}, nil
}

func (h *ArenaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		h.logger.Error("failed to load arena topics", "error", err)
		http.Error(w, "failed to load arena", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "Practice arena", "Modes": game.Modes, "Topics": topics}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render arena", "error", err)
	}
}
