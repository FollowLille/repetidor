package handlers

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
)

type courseView struct {
	domain.LearningCourse
	Depth         int
	Topics        []domain.Topic
	Prerequisites []domain.LearningCourse
}

type CoursesHandler struct {
	templates *template.Template
	courses   storage.LearningCourseRepository
	topics    storage.TopicRepository
	tracks    storage.CourseRepository
	logger    logger.Logger
}

func NewCoursesHandler(courses storage.LearningCourseRepository, topics storage.TopicRepository, tracks storage.CourseRepository, log logger.Logger) (*CoursesHandler, error) {
	tmpl, err := parsePage("courses.html")
	return &CoursesHandler{templates: tmpl, courses: courses, topics: topics, tracks: tracks, logger: log}, err
}

func (h *CoursesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.create(w, r)
		return
	}
	track := activeCourse(h.tracks, r)
	courses, err := h.courses.ListByTrack(r.Context(), track.ID)
	if err != nil {
		http.Error(w, "failed to load courses", 500)
		return
	}
	topics, err := h.topics.ListByCourse(r.Context(), track.ID)
	if err != nil {
		http.Error(w, "failed to load topics", 500)
		return
	}
	views := buildCourseViews(courses, topics)
	data := pageData(r, map[string]any{"Title": "Courses", "Course": track, "Courses": views, "CourseOptions": courses, "Topics": topics})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render courses", "error", err)
	}
}

func (h *CoursesHandler) create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "course name is required", 400)
		return
	}
	track := activeCourse(h.tracks, r)
	course := domain.LearningCourse{LanguageTrackID: track.ID, Name: name, Description: strings.TrimSpace(r.FormValue("description"))}
	course.SortOrder, _ = strconv.Atoi(r.FormValue("sort_order"))
	if id, _ := strconv.ParseInt(r.FormValue("parent_id"), 10, 64); id > 0 {
		course.ParentID = &id
	}
	course.TopicIDs = formIDs(r.Form["topic_ids"])
	course.PrerequisiteIDs = formIDs(r.Form["prerequisite_ids"])
	if _, err := h.courses.Create(r.Context(), course); err != nil {
		h.logger.Error("create learning course", "error", err)
		http.Error(w, "failed to create course", 400)
		return
	}
	http.Redirect(w, r, "/courses", http.StatusSeeOther)
}

func formIDs(values []string) []int64 {
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func buildCourseViews(courses []domain.LearningCourse, topics []domain.Topic) []courseView {
	byID := make(map[int64]domain.LearningCourse, len(courses))
	topicByID := make(map[int64]domain.Topic, len(topics))
	children := make(map[int64][]domain.LearningCourse)
	for _, course := range courses {
		byID[course.ID] = course
		parent := int64(0)
		if course.ParentID != nil {
			parent = *course.ParentID
		}
		children[parent] = append(children[parent], course)
	}
	for _, topic := range topics {
		topicByID[topic.ID] = topic
	}
	for parent := range children {
		sort.SliceStable(children[parent], func(i, j int) bool {
			if children[parent][i].SortOrder == children[parent][j].SortOrder {
				return children[parent][i].ID < children[parent][j].ID
			}
			return children[parent][i].SortOrder < children[parent][j].SortOrder
		})
	}
	views := make([]courseView, 0, len(courses))
	var walk func(int64, int)
	walk = func(parent int64, depth int) {
		for _, course := range children[parent] {
			view := courseView{LearningCourse: course, Depth: depth}
			for _, id := range course.TopicIDs {
				if topic, ok := topicByID[id]; ok {
					view.Topics = append(view.Topics, topic)
				}
			}
			for _, id := range course.PrerequisiteIDs {
				if prerequisite, ok := byID[id]; ok {
					view.Prerequisites = append(view.Prerequisites, prerequisite)
				}
			}
			views = append(views, view)
			walk(course.ID, depth+1)
		}
	}
	walk(0, 0)
	return views
}
