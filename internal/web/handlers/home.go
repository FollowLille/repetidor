package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"repetidor/internal/storage"
)

type HomeHandler struct {
	templates   *template.Template
	topicRepo   storage.TopicRepository
	sessionRepo storage.SessionRepository
}

type ModeLink struct {
	Name string
	URL  string
}

func NewHomeHandler(topicRepo storage.TopicRepository, sessionRepo storage.SessionRepository) (*HomeHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "home.html"))
	if err != nil {
		return nil, err
	}
	return &HomeHandler{templates: tmpl, topicRepo: topicRepo, sessionRepo: sessionRepo}, nil
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load topics", http.StatusInternalServerError)
		return
	}
	recent, err := h.sessionRepo.ListRecent(r.Context(), 20)
	if err != nil {
		http.Error(w, "failed to load sessions", http.StatusInternalServerError)
		return
	}
	active := recent[:0]
	for _, session := range recent {
		if session.Status == "active" {
			active = append(active, session)
		}
	}

	data := map[string]any{
		"Title": "Repetidor",
		"Modes": []ModeLink{
			{Name: "Mixed", URL: "/train/mixed"},
			{Name: "Due", URL: "/train/due"},
			{Name: "Hard", URL: "/train/hard"},
			{Name: "Easy", URL: "/train/easy"},
			{Name: "Spanish to Russian", URL: "/train/spanish-to-russian"},
			{Name: "Russian to Spanish", URL: "/train/russian-to-spanish"},
			{Name: "Random", URL: "/train/random"},
			{Name: "Build letters", URL: "/train/build"},
			{Name: "Type answer", URL: "/train/type"},
		},
		"Topics":         topics,
		"ActiveSessions": active,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
