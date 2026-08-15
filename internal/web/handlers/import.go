package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/importx"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type ImportHandler struct {
	templates *template.Template
	topics    storage.TopicRepository
	words     storage.WordRepository
	courses   storage.CourseRepository
	logger    logger.Logger
}
type importReport struct{ Imported, Duplicates, Invalid int }

func NewImportHandler(topics storage.TopicRepository, words storage.WordRepository, courses storage.CourseRepository, log logger.Logger) (*ImportHandler, error) {
	tmpl, err := parsePage("import.html")
	return &ImportHandler{templates: tmpl, topics: topics, words: words, courses: courses, logger: log}, err
}

func (h *ImportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.process(w, r)
		return
	}
	h.render(w, r, nil, "")
}
func (h *ImportHandler) render(w http.ResponseWriter, r *http.Request, report *importReport, errMsg string) {
	course := activeCourse(h.courses, r)
	topics, err := h.topics.ListByCourse(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load topics", 500)
		return
	}
	data := pageData(r, map[string]any{"Title": "Import words", "Course": course, "Topics": topics, "Report": report, "Error": errMsg, "TargetLanguage": domain.LanguageByCode(course.TargetLanguage), "ReferenceLanguage": domain.LanguageByCode(course.ReferenceLanguage)})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render import", "error", err)
	}
}

func (h *ImportHandler) process(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		h.render(w, r, nil, "The upload is too large or invalid.")
		return
	}
	topicID, _ := strconv.ParseInt(r.FormValue("topic_id"), 10, 64)
	course := activeCourse(h.courses, r)
	topics, _ := h.topics.ListByCourse(r.Context(), course.ID)
	valid := false
	for _, topic := range topics {
		if topic.ID == topicID {
			valid = true
			break
		}
	}
	if !valid {
		h.render(w, r, nil, "Choose a destination topic.")
		return
	}
	rows := importx.ParseText(r.FormValue("words"))
	if file, _, err := r.FormFile("csv_file"); err == nil {
		defer file.Close()
		csvRows, parseErr := importx.ParseCSV(file)
		if parseErr != nil {
			h.render(w, r, nil, "Could not read the CSV file.")
			return
		}
		rows = append(rows, csvRows...)
	}
	if len(rows) == 0 {
		h.render(w, r, nil, "No valid word pairs found.")
		return
	}
	report := &importReport{}
	for _, row := range rows {
		_, err := h.words.Create(r.Context(), domain.Word{TopicID: topicID, Spanish: strings.TrimSpace(row.Source), Russian: strings.TrimSpace(row.Target), Notes: strings.TrimSpace(row.Notes)})
		if errors.Is(err, storage.ErrWordAlreadyExists) {
			report.Duplicates++
			continue
		}
		if err != nil {
			report.Invalid++
			h.logger.Error("import word", "error", err, "line", row.Line)
			continue
		}
		report.Imported++
	}
	h.render(w, r, report, "")
}
