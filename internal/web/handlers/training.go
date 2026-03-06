package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// TrainingHandler renders a training mode stub page.
type TrainingHandler struct {
	templates *template.Template
}

// NewTrainingHandler creates a new TrainingHandler.
func NewTrainingHandler() (*TrainingHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "training.html"),
	)
	if err != nil {
		return nil, err
	}

	return &TrainingHandler{
		templates: tmpl,
	}, nil
}

// ServeHTTP renders the training page.
func (h *TrainingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trainMode := strings.TrimSpace(chi.URLParam(r, "train_mode"))
	if trainMode == "" {
		trainMode = "unknown"
	}

	data := map[string]any{
		"Title":      "Training",
		"PageTitle":  "Training",
		"TrainMode":  trainMode,
		"PageNotice": "This page is a temporary training stub. Real training flow will be added later.",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
