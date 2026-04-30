package handlers

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type TopicWord struct {
	Spanish string
	Russian string
}

type TopicsHandler struct {
	templates       *template.Template
	topicRepository storage.TopicRepository
}

type TopicHandler struct {
	templates       *template.Template
	topicRepository storage.TopicRepository
}

func NewTopicsHandler(topicRepository storage.TopicRepository) (*TopicsHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "topics.html"),
	)
	if err != nil {
		return nil, err
	}

	return &TopicsHandler{
		templates:       tmpl,
		topicRepository: topicRepository,
	}, nil
}

func NewTopicHandler(topicRepository storage.TopicRepository) (*TopicHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "topic_show.html"),
	)
	if err != nil {
		return nil, err
	}

	return &TopicHandler{
		templates:       tmpl,
		topicRepository: topicRepository,
	}, nil
}

func (h *TopicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderTopicsPage(w, r, "", "", "")
	case http.MethodPost:
		h.createTopic(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TopicsHandler) renderTopicsPage(w http.ResponseWriter, r *http.Request, formError string, formName string, formDescription string) {
	topics, err := h.topicRepository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":           "Topics",
		"Topics":          topics,
		"HasTopics":       len(topics) > 0,
		"FormError":       formError,
		"FormName":        formName,
		"FormDescription": formDescription,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *TopicsHandler) createTopic(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		h.renderTopicsPage(w, r, "Topic name is required.", name, description)
		return
	}

	created, err := h.topicRepository.Create(r.Context(), domain.Topic{
		Name:        name,
		Description: description,
	})
	if err != nil {
		h.renderTopicsPage(w, r, "Failed to create topic. It may already exist.", name, description)
		return
	}

	http.Redirect(w, r, "/topics/"+created.Name, http.StatusSeeOther)
}

func (h *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	if topicName == "" {
		http.Error(w, "topic name is required", http.StatusBadRequest)
		return
	}

	topic, err := h.topicRepository.GetByName(r.Context(), topicName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "topic not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":      "Topic",
		"TopicName":  topic.Name,
		"TopicKey":   topic.Name,
		"TopicNotes": topic.Description,
		"Words":      []TopicWord{},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
