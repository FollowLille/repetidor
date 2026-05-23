package handlers

import (
	"errors"
	"fmt"
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
	wordRepo  storage.WordRepository
}

type TopicEditHandler struct {
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

func NewTopicHandler(topicRepo storage.TopicRepository, wordRepo storage.WordRepository) (*TopicHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "topic_show.html"))
	if err != nil {
		return nil, err
	}
	return &TopicHandler{templates: tmpl, topicRepo: topicRepo, wordRepo: wordRepo}, nil
}

func NewTopicEditHandler(topicRepo storage.TopicRepository) (*TopicEditHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "topic_edit.html"))
	if err != nil {
		return nil, err
	}
	return &TopicEditHandler{templates: tmpl, topicRepo: topicRepo}, nil
}

func (h *TopicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderList(w, r, "", nil)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TopicsHandler) renderList(w http.ResponseWriter, r *http.Request, warning string, form map[string]string) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load topics", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "Topics", "Topics": topics, "Warning": warning, "Form": form}
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
		h.renderList(w, r, "Topic name is required.", map[string]string{"Name": name, "Description": description})
		return
	}
	_, err := h.topicRepo.Create(r.Context(), domain.Topic{Name: name, Description: description})
	if err != nil {
		// TODO: replace string check with explicit driver-specific unique violation mapping.
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			h.renderList(w, r, fmt.Sprintf("Topic %q already exists.", name), map[string]string{"Name": name, "Description": description})
			return
		}
		http.Error(w, "failed to create topic", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+name, http.StatusSeeOther)
}

func (h *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderTopicPage(w, r, "", nil)
	case http.MethodPost:
		h.createWord(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TopicHandler) renderTopicPage(w http.ResponseWriter, r *http.Request, errMsg string, form map[string]string) {
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

	words, err := h.wordRepo.ListByTopicID(r.Context(), topic.ID)
	if err != nil {
		http.Error(w, "failed to load words", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Topic",
		"Topic":    topic,
		"Words":    words,
		"Error":    errMsg,
		"WordForm": form,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicHandler) createWord(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	spanish := strings.TrimSpace(r.FormValue("spanish"))
	russian := strings.TrimSpace(r.FormValue("russian"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	form := map[string]string{"Spanish": spanish, "Russian": russian, "Notes": notes}

	if spanish == "" {
		h.renderTopicPage(w, r, "Spanish is required.", form)
		return
	}
	if russian == "" {
		h.renderTopicPage(w, r, "Russian is required.", form)
		return
	}

	_, err = h.wordRepo.Create(r.Context(), domain.Word{TopicID: topic.ID, Spanish: spanish, Russian: russian, Notes: notes})
	if err != nil {
		// TODO: replace string check with explicit driver-specific unique violation mapping.
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			h.renderTopicPage(w, r, fmt.Sprintf("Word %q → %q already exists in this topic.", spanish, russian), form)
			return
		}
		http.Error(w, "failed to create word", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/topics/"+topicName, http.StatusSeeOther)
}

func (h *TopicEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.renderEdit(w, r, "", nil)
	case http.MethodPost:
		h.update(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TopicEditHandler) renderEdit(w http.ResponseWriter, r *http.Request, errMsg string, override map[string]string) {
	topicName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}
	if override != nil {
		topic.Name = override["Name"]
		topic.Description = override["Description"]
	}
	data := map[string]any{"Title": "Edit topic", "Topic": topic, "Error": errMsg, "OriginalName": topicName}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicEditHandler) update(w http.ResponseWriter, r *http.Request) {
	originalName := strings.TrimSpace(chi.URLParam(r, "topic_name"))
	current, err := h.topicRepo.GetByName(r.Context(), originalName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	newName := strings.TrimSpace(r.FormValue("name"))
	newDescription := strings.TrimSpace(r.FormValue("description"))
	if newName == "" {
		h.renderEdit(w, r, "Topic name is required.", map[string]string{"Name": newName, "Description": newDescription})
		return
	}
	_, err = h.topicRepo.Update(r.Context(), domain.Topic{ID: current.ID, Name: newName, Description: newDescription})
	if err != nil {
		// TODO: replace string check with explicit driver-specific unique violation mapping.
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			h.renderEdit(w, r, fmt.Sprintf("Topic %q already exists.", newName), map[string]string{"Name": newName, "Description": newDescription})
			return
		}
		http.Error(w, "failed to update topic", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+newName, http.StatusSeeOther)
}
