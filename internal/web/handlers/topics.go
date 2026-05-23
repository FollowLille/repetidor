package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"repetidor/internal/domain"
	"repetidor/internal/storage"
	"strings"

	"github.com/go-chi/chi/v5"
)

type TopicsHandler struct {
	templates *template.Template
	topicRepo storage.TopicRepository
}

type TopicHandler struct {
	templates *template.Template
	topicRepo storage.TopicRepository
}

func NewTopicsHandler(topicRepo storage.TopicRepository) (*TopicsHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "topics.html"))
	if err != nil {
		return nil, err
	}
	return &TopicsHandler{templates: tmpl, topicRepo: topicRepo}, nil
}

func NewTopicHandler(topicRepo storage.TopicRepository) (*TopicHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "topic_show.html"))
	if err != nil {
		return nil, err
	}
	return &TopicHandler{templates: tmpl, topicRepo: topicRepo}, nil
}

func (h *TopicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderList(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TopicsHandler) renderList(w http.ResponseWriter, r *http.Request) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load topics", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "Topics", "Topics": topics}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicsHandler) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	_, err := h.topicRepo.Create(r.Context(), domain.Topic{Name: name, Description: description})
	if err != nil {
		// TODO: replace string check with explicit driver-specific unique violation mapping.
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			http.Redirect(w, r, "/topics/"+name, http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to create topic", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+name, http.StatusSeeOther)
}

func (h *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	if topicName == "" {
		http.NotFound(w, r)
		return
	}
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "Topic", "Topic": topic}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
