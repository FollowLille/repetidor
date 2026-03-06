package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// TopicWord represents a temporary topic word preview.
type TopicWord struct {
	Spanish string
	Russian string
}

// TopicHandler renders a single topic page stub.
type TopicHandler struct {
	templates *template.Template
}

// NewTopicHandler creates a new TopicHandler.
func NewTopicHandler() (*TopicHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "topic_show.html"),
	)
	if err != nil {
		return nil, err
	}

	return &TopicHandler{
		templates: tmpl,
	}, nil
}

// ServeHTTP renders the topic page.
func (h *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	if topicName == "" {
		topicName = "unknown"
	}

	data := map[string]any{
		"Title":      "Topic",
		"TopicName":  topicName,
		"TopicKey":   topicName,
		"TopicNotes": "This is a temporary topic page. Later it will contain notes, grouped vocabulary, and training actions.",
		"Words": []TopicWord{
			{Spanish: "agua", Russian: "вода"},
			{Spanish: "pan", Russian: "хлеб"},
			{Spanish: "cuchara", Russian: "ложка"},
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
