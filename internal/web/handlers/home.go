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
	Name, URL, Description, Icon string
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
			{Name: "Mixed", URL: "/train/mixed", Icon: "✦", Description: "Adaptive practice based on your progress"},
			{Name: "Due", URL: "/train/due", Icon: "◷", Description: "Words ready for their next review"},
			{Name: "Hard", URL: "/train/hard", Icon: "↗", Description: "Focus on words with recent mistakes"},
			{Name: "Easy", URL: "/train/easy", Icon: "✓", Description: "Reinforce words you already know"},
			{Name: "Spanish → Russian", URL: "/train/spanish-to-russian", Icon: "ES", Description: "Translate from Spanish"},
			{Name: "Russian → Spanish", URL: "/train/russian-to-spanish", Icon: "RU", Description: "Recall the Spanish word"},
			{Name: "Random", URL: "/train/random", Icon: "⤨", Description: "Uniform shuffle across vocabulary"},
			{Name: "Build letters", URL: "/train/build", Icon: "Aa", Description: "Assemble answers letter by letter"},
			{Name: "Type answer", URL: "/train/type", Icon: "⌨", Description: "Practice free recall by typing"},
		},
		"Topics":         topics,
		"ActiveSessions": active,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
