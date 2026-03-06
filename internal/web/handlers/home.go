package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type HomeHandler struct {
	templates *template.Template
}

type ModeLink struct {
	Name string
	URL  string
}

type TopicLink struct {
	Name string
	URL  string
}

func NewHomeHandler() (*HomeHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "home.html"),
	)
	if err != nil {
		return nil, err
	}
	return &HomeHandler{
		templates: tmpl,
	}, nil
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title": "Repetidor",
		"Modes": []ModeLink{
			{Name: "Due", URL: "/train/due"},
			{Name: "Hard", URL: "/train/hard"},
			{Name: "Easy", URL: "/train/easy"},
			{Name: "Random", URL: "/train/random"},
		},
		"Topics": []TopicLink{
			{Name: "Comida", URL: "/topics/comida"},
			{Name: "Trabajo", URL: "/topics/trabajo"},
			{Name: "Viajes", URL: "/topics/viajes"},
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		panic(err)
	}
}
