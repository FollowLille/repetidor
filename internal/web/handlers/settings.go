package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type SettingsHandler struct {
	templates *template.Template
	courses   storage.CourseRepository
	logger    logger.Logger
}

func NewSettingsHandler(courses storage.CourseRepository, log logger.Logger) (*SettingsHandler, error) {
	tmpl, err := parsePage("settings.html")
	return &SettingsHandler{templates: tmpl, courses: courses, logger: log}, err
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.create(w, r)
		return
	}
	courses, err := h.courses.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load courses", 500)
		return
	}
	data := pageData(r, map[string]any{"Title": "Settings", "Courses": courses, "ActiveCourse": activeCourse(h.courses, r), "Languages": domain.Languages})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render settings", "error", err)
	}
}

func (h *SettingsHandler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	target := r.FormValue("target_language")
	reference := r.FormValue("reference_language")
	theory := r.FormValue("theory_language")
	if name == "" || target == "" || reference == "" || target == reference {
		http.Error(w, "invalid course", 400)
		return
	}
	course, err := h.courses.Create(r.Context(), domain.Course{Name: name, TargetLanguage: target, ReferenceLanguage: reference, TheoryLanguage: theory})
	if err != nil {
		h.logger.Error("create course", "error", err)
		http.Error(w, "failed to create course", 500)
		return
	}
	h.setCourse(w, course.ID)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *SettingsHandler) SetLocale(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang != "ru" {
		lang = "en"
	}
	http.SetCookie(w, &http.Cookie{Name: "repetidor_locale", Value: lang, Path: "/", Expires: time.Now().AddDate(1, 0, 0), SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, safeNext(r.URL.Query().Get("next")), http.StatusSeeOther)
}
func (h *SettingsHandler) SetCourse(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("track_id"), 10, 64)
	if id == 0 {
		id, _ = strconv.ParseInt(r.FormValue("course_id"), 10, 64)
	}
	if _, err := h.courses.Get(r.Context(), id); err != nil {
		http.Error(w, "course not found", 404)
		return
	}
	h.setCourse(w, id)
	http.Redirect(w, r, safeNext(r.FormValue("next")), http.StatusSeeOther)
}
func (h *SettingsHandler) setCourse(w http.ResponseWriter, id int64) {
	http.SetCookie(w, &http.Cookie{Name: "repetidor_track", Value: strconv.FormatInt(id, 10), Path: "/", Expires: time.Now().AddDate(1, 0, 0), SameSite: http.SameSiteLaxMode})
}
