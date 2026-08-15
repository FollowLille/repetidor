package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

type HomeHandler struct {
	templates   *template.Template
	topicRepo   storage.TopicRepository
	sessionRepo storage.SessionRepository
	courseRepo  storage.CourseRepository
}
type ModeLink struct{ Name, URL, Description, Icon string }

func NewHomeHandler(topics storage.TopicRepository, sessions storage.SessionRepository, courses storage.CourseRepository) (*HomeHandler, error) {
	tmpl, err := parsePage("home.html")
	if err != nil {
		return nil, err
	}
	return &HomeHandler{templates: tmpl, topicRepo: topics, sessionRepo: sessions, courseRepo: courses}, nil
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	course := activeCourse(h.courseRepo, r)
	topics, err := h.topicRepo.ListByCourse(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load topics", 500)
		return
	}
	recent, err := h.sessionRepo.ListRecent(r.Context(), 20)
	if err != nil {
		http.Error(w, "failed to load sessions", 500)
		return
	}
	active := recent[:0]
	for _, session := range recent {
		if session.Status == "active" {
			active = append(active, session)
		}
	}
	target, reference := domain.LanguageByCode(course.TargetLanguage), domain.LanguageByCode(course.ReferenceLanguage)
	uiLocale := locale(r)
	data := pageData(r, map[string]any{"Title": "Repetidor", "Course": course, "TargetLanguage": target, "ReferenceLanguage": reference, "Modes": []ModeLink{
		{Name: "Mixed", URL: "/train/mixed", Icon: "✦", Description: "Adaptive practice based on your progress"}, {Name: "Due", URL: "/train/due", Icon: "◷", Description: "Words ready for their next review"}, {Name: "Hard", URL: "/train/hard", Icon: "↗", Description: "Focus on words with recent mistakes"}, {Name: "Easy", URL: "/train/easy", Icon: "✓", Description: "Reinforce words you already know"},
		{Name: target.NativeName + " → " + reference.NativeName, URL: "/train/spanish-to-russian", Icon: strings.ToUpper(target.Code), Description: translate(uiLocale, "Translate into") + " " + reference.NativeName}, {Name: reference.NativeName + " → " + target.NativeName, URL: "/train/russian-to-spanish", Icon: strings.ToUpper(reference.Code), Description: translate(uiLocale, "Recall the word in") + " " + target.NativeName},
		{Name: "Random", URL: "/train/random", Icon: "⤨", Description: "Uniform shuffle across vocabulary"}, {Name: "Build letters", URL: "/train/build", Icon: "Aa", Description: "Assemble answers letter by letter"}, {Name: "Type answer", URL: "/train/type", Icon: "⌨", Description: "Practice free recall by typing"}}, "Topics": topics, "ActiveSessions": active})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
