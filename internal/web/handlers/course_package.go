package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"repetidor/internal/coursepack"
	"repetidor/internal/logger"
	"repetidor/internal/storage"

	"github.com/go-chi/chi/v5"
)

type CoursePackageHandler struct {
	templates *template.Template
	packages  storage.CoursePackageRepository
	courses   storage.CourseRepository
	logger    logger.Logger
}

func NewCoursePackageHandler(packages storage.CoursePackageRepository, courses storage.CourseRepository, log logger.Logger) (*CoursePackageHandler, error) {
	tmpl, err := parsePage("course_import.html")
	return &CoursePackageHandler{templates: tmpl, packages: packages, courses: courses, logger: log}, err
}

func (h *CoursePackageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.process(w, r)
		return
	}
	h.render(w, r, nil, "", "")
}

func (h *CoursePackageHandler) render(w http.ResponseWriter, r *http.Request, summary *storage.CoursePackageSummary, packageJSON, errorMessage string) {
	course := activeCourse(h.courses, r)
	data := pageData(r, map[string]any{"Title": "Import course", "Course": course, "Summary": summary, "PackageJSON": packageJSON, "Error": errorMessage})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render course import", "error", err)
	}
}

func (h *CoursePackageHandler) process(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		h.render(w, r, nil, "", "The course file is too large or invalid.")
		return
	}
	raw := []byte(strings.TrimSpace(r.FormValue("package_json")))
	if file, _, err := r.FormFile("course_file"); err == nil {
		defer file.Close()
		raw, err = io.ReadAll(io.LimitReader(file, 10<<20))
		if err != nil {
			h.render(w, r, nil, "", "Could not read the course file.")
			return
		}
	}
	value, err := coursepack.Decode(raw)
	if err != nil {
		h.render(w, r, nil, "", err.Error())
		return
	}
	canonical, _ := json.MarshalIndent(value, "", "  ")
	track := activeCourse(h.courses, r)
	if r.FormValue("action") == "import" {
		id, _, importErr := h.packages.Import(r.Context(), track.ID, value)
		if importErr == storage.ErrCoursePackageDuplicate && id > 0 {
			http.Redirect(w, r, "/courses/"+strconv.FormatInt(id, 10)+"?import=duplicate", http.StatusSeeOther)
			return
		}
		if importErr != nil {
			h.render(w, r, nil, string(canonical), importErr.Error())
			return
		}
		http.Redirect(w, r, "/courses/"+strconv.FormatInt(id, 10)+"?import=success", http.StatusSeeOther)
		return
	}
	summary, err := h.packages.Preview(r.Context(), track.ID, value)
	if err != nil {
		h.render(w, r, nil, string(canonical), err.Error())
		return
	}
	h.render(w, r, &summary, string(canonical), "")
}

func (h *CoursePackageHandler) Export(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "course_id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid course", 400)
		return
	}
	value, err := h.packages.Export(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not export this course.", 500)
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		http.Error(w, "Could not encode this course.", 500)
		return
	}
	filename := "course-" + strconv.FormatInt(id, 10) + ".repetidor.json"
	displayName := strings.NewReplacer("/", "-", "\\", "-").Replace(value.Course.Name) + ".repetidor.json"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, url.PathEscape(displayName)))
	_, _ = w.Write(data)
}
