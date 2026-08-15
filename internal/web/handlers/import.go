package handlers

import (
	"encoding/csv"
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
	templates       *template.Template
	topics          storage.TopicRepository
	words           storage.WordRepository
	courses         storage.CourseRepository
	logger          logger.Logger
	learningCourses storage.LearningCourseRepository
}
type importReport struct{ Imported, Duplicates, Invalid int }

func NewImportHandler(topics storage.TopicRepository, words storage.WordRepository, courses storage.CourseRepository, learningCourses storage.LearningCourseRepository, log logger.Logger) (*ImportHandler, error) {
	tmpl, err := parsePage("import.html")
	return &ImportHandler{templates: tmpl, topics: topics, words: words, courses: courses, learningCourses: learningCourses, logger: log}, err
}

func (h *ImportHandler) Sample(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="repetidor-vocabulary-sample.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"topic", "target", "reference", "notes"})
	_ = writer.Write([]string{"Food", "carne", "мясо", ""})
	_ = writer.Write([]string{"Food", "pan", "хлеб", ""})
	_ = writer.Write([]string{"Kitchen", "cuchara", "ложка", "feminine"})
	writer.Flush()
}

func (h *ImportHandler) Export(w http.ResponseWriter, r *http.Request) {
	track := activeCourse(h.courses, r)
	topics, err := h.topics.ListByCourse(r.Context(), track.ID)
	if err != nil {
		http.Error(w, "Could not export vocabulary.", 500)
		return
	}
	scope := r.URL.Query().Get("scope")
	allowed := map[int64]bool{}
	if scope == "topic" {
		id, _ := strconv.ParseInt(r.URL.Query().Get("topic_id"), 10, 64)
		allowed[id] = true
	} else if scope == "course" {
		id, _ := strconv.ParseInt(r.URL.Query().Get("course_id"), 10, 64)
		course, getErr := h.learningCourses.Get(r.Context(), id)
		if getErr != nil || course.LanguageTrackID != track.ID {
			http.Error(w, "Course not found.", 404)
			return
		}
		for _, topicID := range course.TopicIDs {
			allowed[topicID] = true
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="repetidor-vocabulary.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"topic", "target", "reference", "notes"})
	for _, topic := range topics {
		if len(allowed) > 0 && !allowed[topic.ID] {
			continue
		}
		words, listErr := h.words.ListByTopicID(r.Context(), topic.ID)
		if listErr != nil {
			continue
		}
		for _, word := range words {
			_ = writer.Write([]string{topic.Name, word.Spanish, word.Russian, word.Notes})
		}
	}
	writer.Flush()
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
	learningCourses, _ := h.learningCourses.ListByTrack(r.Context(), course.ID)
	data := pageData(r, map[string]any{"Title": "Import words", "Course": course, "Topics": topics, "LearningCourses": learningCourses, "Report": report, "Error": errMsg, "TargetLanguage": domain.LanguageByCode(course.TargetLanguage), "ReferenceLanguage": domain.LanguageByCode(course.ReferenceLanguage)})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render import", "error", err)
	}
}

func (h *ImportHandler) process(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		h.render(w, r, nil, "The upload is too large or invalid.")
		return
	}
	topicID, _ := strconv.ParseInt(r.FormValue("topic_id"), 10, 64)
	newTopic := strings.TrimSpace(r.FormValue("new_topic"))
	course := activeCourse(h.courses, r)
	topics, _ := h.topics.ListByCourse(r.Context(), course.ID)
	valid := false
	for _, topic := range topics {
		if topic.ID == topicID {
			valid = true
			break
		}
	}
	if !valid && newTopic != "" {
		created, err := h.topics.Create(r.Context(), domain.Topic{CourseID: course.ID, Name: newTopic, Description: "Imported vocabulary"})
		if errors.Is(err, storage.ErrTopicAlreadyExists) {
			existing, getErr := h.topics.GetByName(r.Context(), newTopic)
			if getErr == nil && existing.CourseID == course.ID {
				topicID, valid = existing.ID, true
			}
		} else if err != nil {
			h.logger.Error("create import topic", "error", err)
			h.render(w, r, nil, "Could not create the destination topic.")
			return
		} else {
			topicID, valid = created.ID, true
		}
	}
	if !valid {
		h.render(w, r, nil, "Choose or create a destination topic.")
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
	if file, _, err := r.FormFile("excel_file"); err == nil {
		defer file.Close()
		excelRows, parseErr := importx.ParseExcel(file)
		if parseErr != nil {
			h.render(w, r, nil, "Could not read the Excel file.")
			return
		}
		rows = append(rows, excelRows...)
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
