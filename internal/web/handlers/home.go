package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type HomeHandler struct {
	templates *template.Template
}

func NewHomeHandler() (*HomeHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "home.html"),
		filepath.Join("web", "templates", "layout.html"),
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
		"Title":  "Repetidor",
		"Modes":  []string{"Due", "Hard", "Easy", "Random"},
		"Topics": []string{"Comida", "Trabajo", "Viajes"},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		panic(err)
	}
}
