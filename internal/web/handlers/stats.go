package handlers

import (
	"html/template"
	"net/http"

	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type StatsHandler struct {
	templates    *template.Template
	trainingRepo storage.TrainingRepository
	sessionRepo  storage.SessionRepository
	logger       logger.Logger
}

func NewStatsHandler(trainingRepo storage.TrainingRepository, sessionRepo storage.SessionRepository, appLogger logger.Logger) (*StatsHandler, error) {
	tmpl, err := parsePage("stats.html")
	if err != nil {
		return nil, err
	}
	return &StatsHandler{templates: tmpl, trainingRepo: trainingRepo, sessionRepo: sessionRepo, logger: appLogger}, nil
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stats, err := h.trainingRepo.ListStats(r.Context())
	if err != nil {
		h.logger.Error("failed to load training stats", "error", err)
		http.Error(w, "failed to load training stats", http.StatusInternalServerError)
		return
	}
	sessions, err := h.sessionRepo.ListRecent(r.Context(), 10)
	if err != nil {
		h.logger.Error("failed to load training sessions", "error", err)
		http.Error(w, "failed to load training sessions", http.StatusInternalServerError)
		return
	}
	mistakes, err := h.sessionRepo.ListFrequentMistakes(r.Context(), 8)
	if err != nil {
		h.logger.Error("failed to load frequent mistakes", "error", err)
		http.Error(w, "failed to load frequent mistakes", http.StatusInternalServerError)
		return
	}

	totals := struct {
		Words, Seen, Correct, Wrong, Accuracy int
	}{Words: len(stats)}
	for _, item := range stats {
		totals.Seen += item.SeenCount
		totals.Correct += item.CorrectCount
		totals.Wrong += item.WrongCount
	}
	if totals.Seen > 0 {
		totals.Accuracy = totals.Correct * 100 / totals.Seen
	}

	data := pageData(r, map[string]any{"Title": "Training statistics", "Stats": stats, "Totals": totals, "Sessions": sessions, "FrequentMistakes": mistakes})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render training stats", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
