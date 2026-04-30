package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type HomeHandler struct {
	templates       *template.Template
	topicRepository storage.TopicRepository
}

type ModeLink struct {
	Name string
	URL  string
}

type TopicLink struct {
	Name string
	URL  string
}

func NewHomeHandler(topicRepository storage.TopicRepository) (*HomeHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "home.html"),
	)
	if err != nil {
		return nil, err
	}
	return &HomeHandler{
		templates:       tmpl,
		topicRepository: topicRepository,
	}, nil
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, err := h.topicRepository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title": "Repetidor",
		"Modes": []ModeLink{
			{Name: "Due", URL: "/train/due"},
			{Name: "Hard", URL: "/train/hard"},
			{Name: "Easy", URL: "/train/easy"},
			{Name: "Random", URL: "/train/random"},
		},
		"Topics":    buildTopicLinks(topics),
		"HasTopics": len(topics) > 0,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func buildTopicLinks(topics []domain.Topic) []TopicLink {
	result := make([]TopicLink, 0, len(topics))

	for _, topic := range topics {
		result = append(result, TopicLink{
			Name: topic.Name,
			URL:  "/topics/" + topic.Name,
		})
	}

	return result
}
