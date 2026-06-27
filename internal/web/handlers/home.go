package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"repetidor/internal/storage"
)

type HomeHandler struct {
	templates *template.Template
	topicRepo storage.TopicRepository
}

type ModeLink struct {
	Name string
	URL  string
}

func NewHomeHandler(topicRepo storage.TopicRepository) (*HomeHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "home.html"))
	if err != nil {
		return nil, err
	}
	return &HomeHandler{templates: tmpl, topicRepo: topicRepo}, nil
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load topics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title": "Repetidor",
		"Modes": []ModeLink{
			{Name: "Mixed", URL: "/train/mixed"},
			{Name: "Letters", URL: "/train/build"},
			{Name: "Typing", URL: "/train/type"},
			{Name: "Spanish to Russian", URL: "/train/spanish-to-russian"},
			{Name: "Russian to Spanish", URL: "/train/russian-to-spanish"},
		},
		"Topics": topics,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
