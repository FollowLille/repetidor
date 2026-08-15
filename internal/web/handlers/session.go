package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"

	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"

	"github.com/go-chi/chi/v5"
)

type SessionHandler struct {
	templates *template.Template
	repo      storage.SessionRepository
	logger    logger.Logger
}

type sessionCardView struct {
	Position                                                         int
	TopicName, Direction, Prompt, Target, Response, State, ErrorKind string
	EditDistance                                                     int
	Answered, Correct                                                bool
}

func NewSessionHandler(repo storage.SessionRepository, appLogger logger.Logger) (*SessionHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "session.html"))
	if err != nil {
		return nil, err
	}
	return &SessionHandler{templates: tmpl, repo: repo, logger: appLogger}, nil
}

func (h *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "session_id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid session", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodPost {
		if err := h.repo.Abandon(r.Context(), id); err != nil {
			h.logger.Error("failed to abandon session", "error", err)
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Redirect(w, r, "/stats/sessions/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
		return
	}
	session, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	cards, err := h.repo.ListCards(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to load session cards", "error", err)
		http.Error(w, "failed to load session cards", http.StatusInternalServerError)
		return
	}
	views := make([]sessionCardView, 0, len(cards))
	for _, card := range cards {
		prompt, target := promptAndTarget(card.Word, card.Direction)
		state, response := "Pending", card.Response
		if card.Answered {
			state = "Wrong"
			if card.Correct {
				state = "Correct"
			} else if card.ErrorKind == domain.AnswerSkipped {
				state = "Skipped"
			} else if card.ErrorKind == domain.AnswerDontKnow {
				state = "Don't know"
			}
			if response == "" {
				if card.ErrorKind == domain.AnswerDontKnow {
					response = "Don't know"
				} else {
					response = "Skipped"
				}
			}
		}
		views = append(views, sessionCardView{Position: card.Position, TopicName: card.Topic.Name, Direction: labelDirection(card.Direction), Prompt: prompt, Target: target, Response: response, State: state, ErrorKind: card.ErrorKind, EditDistance: card.EditDistance, Answered: card.Answered, Correct: card.Correct})
	}
	data := map[string]any{"Title": "Session details", "Session": session, "Cards": views, "Active": session.Status == domain.SessionActive, "ResumePath": sessionURL(session.Mode, session.ID)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render session details", "error", err)
	}
}
