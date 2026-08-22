package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func topicNameParam(r *http.Request) string {
	raw := chi.URLParam(r, "topic_name")
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	return strings.TrimSpace(raw)
}

type TopicsHandler struct {
	templates  *template.Template
	topicRepo  storage.TopicRepository
	courseRepo storage.CourseRepository
	logger     logger.Logger
}

type TopicHandler struct {
	templates  *template.Template
	topicRepo  storage.TopicRepository
	wordRepo   storage.WordRepository
	courseRepo storage.CourseRepository
	logger     logger.Logger
}

type TopicEditHandler struct {
	templates *template.Template
	topicRepo storage.TopicRepository
	logger    logger.Logger
}

type WordEditHandler struct {
	templates  *template.Template
	topicRepo  storage.TopicRepository
	wordRepo   storage.WordRepository
	courseRepo storage.CourseRepository
	logger     logger.Logger
}

func NewTopicsHandler(topicRepo storage.TopicRepository, courseRepo storage.CourseRepository, appLogger logger.Logger) (*TopicsHandler, error) {
	tmpl, err := parsePage("topics.html")
	if err != nil {
		return nil, err
	}
	return &TopicsHandler{templates: tmpl, topicRepo: topicRepo, courseRepo: courseRepo, logger: appLogger}, nil
}

func NewTopicHandler(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, courseRepo storage.CourseRepository, appLogger logger.Logger) (*TopicHandler, error) {
	tmpl, err := parsePage("topic_show.html")
	if err != nil {
		return nil, err
	}
	return &TopicHandler{templates: tmpl, topicRepo: topicRepo, wordRepo: wordRepo, courseRepo: courseRepo, logger: appLogger}, nil
}

func NewTopicEditHandler(topicRepo storage.TopicRepository, appLogger logger.Logger) (*TopicEditHandler, error) {
	tmpl, err := parsePage("topic_edit.html")
	if err != nil {
		return nil, err
	}
	return &TopicEditHandler{templates: tmpl, topicRepo: topicRepo, logger: appLogger}, nil
}

func NewWordEditHandler(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, courseRepo storage.CourseRepository, appLogger logger.Logger) (*WordEditHandler, error) {
	tmpl, err := parsePage("word_edit.html")
	if err != nil {
		return nil, err
	}
	return &WordEditHandler{templates: tmpl, topicRepo: topicRepo, wordRepo: wordRepo, courseRepo: courseRepo, logger: appLogger}, nil
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
	topics, err := h.topicRepo.ListByCourse(r.Context(), activeCourse(h.courseRepo, r).ID)
	if err != nil {
		h.logger.Error("failed to load topics", "error", err)
		http.Error(w, "failed to load topics", http.StatusInternalServerError)
		return
	}
	data := pageData(r, map[string]any{"Title": "Topics", "Topics": topics, "Warning": warning, "Form": form, "Course": activeCourse(h.courseRepo, r)})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render topics page", "error", err)
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
		h.renderList(w, r, requiredMessage(r, translate(locale(r), "Name")), map[string]string{"Name": name, "Description": description})
		return
	}
	_, err := h.topicRepo.Create(r.Context(), domain.Topic{CourseID: activeCourse(h.courseRepo, r).ID, Name: name, Description: description})
	if err != nil {
		if errors.Is(err, storage.ErrTopicAlreadyExists) {
			h.renderList(w, r, topicExistsMessage(r, name), map[string]string{"Name": name, "Description": description})
			return
		}
		h.logger.Error("failed to create topic", "error", err, "topic_name", name)
		http.Error(w, "failed to create topic", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+name, http.StatusSeeOther)
}

func (h *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.renderShow(w, r, "", nil)
}

func (h *TopicHandler) renderShow(w http.ResponseWriter, r *http.Request, warning string, form map[string]string) {
	topicName := topicNameParam(r)
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
		h.logger.Error("failed to load topic", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}

	words, err := h.wordRepo.ListByTopicID(r.Context(), topic.ID)
	if err != nil {
		h.logger.Error("failed to load words", "error", err, "topic_id", topic.ID)
		http.Error(w, "failed to load words", http.StatusInternalServerError)
		return
	}

	course, _ := h.courseRepo.Get(r.Context(), topic.CourseID)
	data := pageData(r, map[string]any{"Title": "Topic", "Topic": topic, "Words": words, "Warning": warning, "Form": form, "Course": course, "TargetLanguage": domain.LanguageByCode(course.TargetLanguage), "ReferenceLanguage": domain.LanguageByCode(course.ReferenceLanguage)})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render topic page", "error", err, "topic_id", topic.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicHandler) CreateWord(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	spanish := strings.TrimSpace(r.FormValue("spanish"))
	russian := strings.TrimSpace(r.FormValue("russian"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	form := map[string]string{"Spanish": spanish, "Russian": russian, "Notes": notes}

	if spanish == "" {
		h.renderShow(w, r, requiredMessage(r, domain.LanguageByCode(activeCourse(h.courseRepo, r).TargetLanguage).NativeName), form)
		return
	}
	if russian == "" {
		h.renderShow(w, r, requiredMessage(r, domain.LanguageByCode(activeCourse(h.courseRepo, r).ReferenceLanguage).NativeName), form)
		return
	}

	topicName := topicNameParam(r)
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to load topic for word creation", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}

	_, err = h.wordRepo.Create(r.Context(), domain.Word{TopicID: topic.ID, Spanish: spanish, Russian: russian, Notes: notes})
	if err != nil {
		if errors.Is(err, storage.ErrWordAlreadyExists) {
			h.renderShow(w, r, wordExistsMessage(r, spanish, russian), form)
			return
		}
		h.logger.Error("failed to create word", "error", err, "topic_id", topic.ID)
		http.Error(w, "failed to create word", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/topics/"+topic.Name, http.StatusSeeOther)
}

func (h *TopicHandler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	topicName := topicNameParam(r)
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to load topic for word deletion", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}

	wordID, err := strconv.ParseInt(chi.URLParam(r, "word_id"), 10, 64)
	if err != nil || wordID <= 0 {
		http.NotFound(w, r)
		return
	}

	err = h.wordRepo.Delete(r.Context(), topic.ID, wordID)
	if errors.Is(err, storage.ErrWordNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to delete word", "error", err, "topic_id", topic.ID, "word_id", wordID)
		http.Error(w, "failed to delete word", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/topics/"+topic.Name, http.StatusSeeOther)
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
	topicName := topicNameParam(r)
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to load topic for edit", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}
	if override != nil {
		topic.Name = override["Name"]
		topic.Description = override["Description"]
	}
	data := pageData(r, map[string]any{"Title": "Edit topic", "Topic": topic, "Error": errMsg, "OriginalName": topicName})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render topic edit page", "error", err, "topic_id", topic.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicEditHandler) update(w http.ResponseWriter, r *http.Request) {
	originalName := topicNameParam(r)
	current, err := h.topicRepo.GetByName(r.Context(), originalName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to load topic for update", "error", err, "topic_name", originalName)
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
		h.renderEdit(w, r, requiredMessage(r, translate(locale(r), "Name")), map[string]string{"Name": newName, "Description": newDescription})
		return
	}
	_, err = h.topicRepo.Update(r.Context(), domain.Topic{ID: current.ID, Name: newName, Description: newDescription})
	if err != nil {
		if errors.Is(err, storage.ErrTopicAlreadyExists) {
			h.renderEdit(w, r, topicExistsMessage(r, newName), map[string]string{"Name": newName, "Description": newDescription})
			return
		}
		h.logger.Error("failed to update topic", "error", err, "topic_id", current.ID, "new_name", newName)
		http.Error(w, "failed to update topic", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+newName, http.StatusSeeOther)
}

func (h *TopicEditHandler) Delete(w http.ResponseWriter, r *http.Request) {
	topicName := topicNameParam(r)
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.logger.Error("failed to load topic for deletion", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return
	}

	if err := h.topicRepo.Delete(r.Context(), topic.ID); errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		h.logger.Error("failed to delete topic", "error", err, "topic_id", topic.ID)
		http.Error(w, "failed to delete topic", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/topics", http.StatusSeeOther)
}

func (h *WordEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.render(w, r, "", nil)
	case http.MethodPost:
		h.update(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WordEditHandler) load(w http.ResponseWriter, r *http.Request) (domain.Topic, domain.Word, bool) {
	topicName := topicNameParam(r)
	topic, err := h.topicRepo.GetByName(r.Context(), topicName)
	if errors.Is(err, storage.ErrTopicNotFound) {
		http.NotFound(w, r)
		return domain.Topic{}, domain.Word{}, false
	}
	if err != nil {
		h.logger.Error("failed to load topic for word edit", "error", err, "topic_name", topicName)
		http.Error(w, "failed to load topic", http.StatusInternalServerError)
		return domain.Topic{}, domain.Word{}, false
	}

	wordID, err := strconv.ParseInt(chi.URLParam(r, "word_id"), 10, 64)
	if err != nil || wordID <= 0 {
		http.NotFound(w, r)
		return domain.Topic{}, domain.Word{}, false
	}
	word, err := h.wordRepo.GetByID(r.Context(), topic.ID, wordID)
	if errors.Is(err, storage.ErrWordNotFound) {
		http.NotFound(w, r)
		return domain.Topic{}, domain.Word{}, false
	}
	if err != nil {
		h.logger.Error("failed to load word for edit", "error", err, "topic_id", topic.ID, "word_id", wordID)
		http.Error(w, "failed to load word", http.StatusInternalServerError)
		return domain.Topic{}, domain.Word{}, false
	}
	return topic, word, true
}

func (h *WordEditHandler) render(w http.ResponseWriter, r *http.Request, errMsg string, override map[string]string) {
	topic, word, ok := h.load(w, r)
	if !ok {
		return
	}
	if override != nil {
		word.Spanish = override["Spanish"]
		word.Russian = override["Russian"]
		word.Notes = override["Notes"]
	}
	course, _ := h.courseRepo.Get(r.Context(), topic.CourseID)
	data := pageData(r, map[string]any{"Title": "Edit word", "Topic": topic, "Word": word, "Error": errMsg, "Course": course, "TargetLanguage": domain.LanguageByCode(course.TargetLanguage), "ReferenceLanguage": domain.LanguageByCode(course.ReferenceLanguage)})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render word edit page", "error", err, "word_id", word.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *WordEditHandler) update(w http.ResponseWriter, r *http.Request) {
	topic, word, ok := h.load(w, r)
	if !ok {
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
		h.render(w, r, requiredMessage(r, domain.LanguageByCode(activeCourse(h.courseRepo, r).TargetLanguage).NativeName), form)
		return
	}
	if russian == "" {
		h.render(w, r, requiredMessage(r, domain.LanguageByCode(activeCourse(h.courseRepo, r).ReferenceLanguage).NativeName), form)
		return
	}
	word.Spanish = spanish
	word.Russian = russian
	word.Notes = notes
	if _, err := h.wordRepo.Update(r.Context(), word); errors.Is(err, storage.ErrWordAlreadyExists) {
		h.render(w, r, wordExistsMessage(r, spanish, russian), form)
		return
	} else if errors.Is(err, storage.ErrWordNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		h.logger.Error("failed to update word", "error", err, "topic_id", topic.ID, "word_id", word.ID)
		http.Error(w, "failed to update word", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topics/"+topic.Name, http.StatusSeeOther)
}
