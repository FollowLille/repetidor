package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"

	"github.com/go-chi/chi/v5"
)

type CourseHandler struct {
	pageTemplates     *template.Template
	practiceTemplates *template.Template
	courses           storage.LearningCourseRepository
	theory            storage.TheoryRepository
	tracks            storage.CourseRepository
	topics            storage.TopicRepository
	boards            storage.BoardRepository
	logger            logger.Logger
}

type theorySection struct {
	Block     domain.TheoryBlock
	Exercises []domain.TheoryExercise
}

func NewCourseHandler(courses storage.LearningCourseRepository, theory storage.TheoryRepository, tracks storage.CourseRepository, topics storage.TopicRepository, boards storage.BoardRepository, log logger.Logger) (*CourseHandler, error) {
	page, err := parsePage("course_show.html")
	if err != nil {
		return nil, err
	}
	practice, err := parsePage("course_practice.html")
	return &CourseHandler{pageTemplates: page, practiceTemplates: practice, courses: courses, theory: theory, tracks: tracks, topics: topics, boards: boards, logger: log}, err
}

func (h *CourseHandler) course(r *http.Request) (domain.LearningCourse, bool) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "course_id"), 10, 64)
	course, err := h.courses.Get(r.Context(), id)
	return course, err == nil && course.LanguageTrackID == activeCourse(h.tracks, r).ID
}

func (h *CourseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	blocks, err := h.theory.ListBlocks(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load theory", 500)
		return
	}
	exercises, err := h.theory.ListExercises(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load exercises", 500)
		return
	}
	sections := make([]theorySection, 0, len(blocks))
	for _, block := range blocks {
		section := theorySection{Block: block}
		for _, exercise := range exercises {
			if exercise.TheoryBlockID != nil && *exercise.TheoryBlockID == block.ID {
				section.Exercises = append(section.Exercises, exercise)
			}
		}
		sections = append(sections, section)
	}
	var courseExercises []domain.TheoryExercise
	for _, exercise := range exercises {
		if exercise.TheoryBlockID == nil {
			courseExercises = append(courseExercises, exercise)
		}
	}
	progress, err := h.theory.Progress(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load progress", 500)
		return
	}
	track := activeCourse(h.tracks, r)
	courseOptions, _ := h.courses.ListByTrack(r.Context(), track.ID)
	levels, _ := h.courses.ListLevels(r.Context(), track.ID)
	topics, _ := h.topics.ListByCourse(r.Context(), track.ID)
	boards, _ := h.boards.ListByCourse(r.Context(), course.ID)
	query := url.Values{}
	for _, id := range course.TopicIDs {
		query.Add("topic_ids", strconv.FormatInt(id, 10))
	}
	practiceURL := "/train/mixed"
	if encoded := query.Encode(); encoded != "" {
		practiceURL += "?" + encoded
	}
	data := pageData(r, map[string]any{"Title": course.Name, "Course": track, "LearningCourse": course, "Blocks": blocks, "BlockSections": sections, "Exercises": exercises, "CourseExercises": courseExercises, "Progress": progress, "CourseOptions": courseOptions, "Topics": topics, "PracticeURL": practiceURL, "Boards": boards, "Levels": levels})
	if err := h.pageTemplates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render course", "error", err)
	}
}

func (h *CourseHandler) Update(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "course name is required", 400)
		return
	}
	course.Name = name
	course.Description = strings.TrimSpace(r.FormValue("description"))
	course.SortOrder, _ = strconv.Atoi(r.FormValue("sort_order"))
	course.ParentID = nil
	if id, _ := strconv.ParseInt(r.FormValue("parent_id"), 10, 64); id > 0 && id != course.ID {
		course.ParentID = &id
	}
	course.TopicIDs = formIDs(r.Form["topic_ids"])
	course.PrerequisiteIDs = formIDs(r.Form["prerequisite_ids"])
	course.LevelIDs = formIDs(r.Form["level_ids"])
	if _, err := h.courses.Update(r.Context(), course); err != nil {
		h.logger.Error("update course", "error", err)
		http.Error(w, "failed to update course", 400)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10), http.StatusSeeOther)
}

func (h *CourseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.courses.Delete(r.Context(), course.ID); err != nil {
		http.Error(w, "failed to delete course", 500)
		return
	}
	http.Redirect(w, r, "/courses", http.StatusSeeOther)
}

func (h *CourseHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "block_id"), 10, 64)
	blocks, _ := h.theory.ListBlocks(r.Context(), course.ID)
	found := false
	for _, block := range blocks {
		if block.ID == id {
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if err := h.theory.DeleteBlock(r.Context(), id); err != nil {
		http.Error(w, "failed to delete block", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10), http.StatusSeeOther)
}

func (h *CourseHandler) DeleteExercise(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "exercise_id"), 10, 64)
	items, _ := h.theory.ListExercises(r.Context(), course.ID)
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if err := h.theory.DeleteExercise(r.Context(), id); err != nil {
		http.Error(w, "failed to delete exercise", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10), http.StatusSeeOther)
}

func (h *CourseHandler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	kind := r.FormValue("kind")
	if kind != "text" && kind != "example" && kind != "note" && kind != "table" {
		kind = "text"
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "content is required", 400)
		return
	}
	order, _ := strconv.Atoi(r.FormValue("sort_order"))
	_, err := h.theory.CreateBlock(r.Context(), domain.TheoryBlock{CourseID: course.ID, Kind: kind, Title: strings.TrimSpace(r.FormValue("title")), Content: content, SortOrder: order})
	if err != nil {
		http.Error(w, "failed to create theory block", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10), http.StatusSeeOther)
}

func (h *CourseHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	kind := r.FormValue("kind")
	if kind != "choice" && kind != "input" && kind != "gap" {
		kind = "input"
	}
	prompt := strings.TrimSpace(r.FormValue("prompt"))
	answer := strings.TrimSpace(r.FormValue("correct_answer"))
	if prompt == "" || answer == "" {
		http.Error(w, "prompt and answer are required", 400)
		return
	}
	order, _ := strconv.Atoi(r.FormValue("sort_order"))
	var blockID *int64
	if id, _ := strconv.ParseInt(r.FormValue("theory_block_id"), 10, 64); id > 0 {
		blocks, _ := h.theory.ListBlocks(r.Context(), course.ID)
		for _, block := range blocks {
			if block.ID == id {
				value := id
				blockID = &value
				break
			}
		}
		if blockID == nil {
			http.Error(w, "theory block does not belong to course", 400)
			return
		}
	}
	var options []string
	for _, option := range strings.Split(r.FormValue("options"), ",") {
		if value := strings.TrimSpace(option); value != "" {
			options = append(options, value)
		}
	}
	var acceptedAnswers []string
	for _, accepted := range strings.Split(r.FormValue("accepted_answers"), ",") {
		if value := strings.TrimSpace(accepted); value != "" {
			acceptedAnswers = append(acceptedAnswers, value)
		}
	}
	_, err := h.theory.CreateExercise(r.Context(), domain.TheoryExercise{CourseID: course.ID, TheoryBlockID: blockID, Kind: kind, Prompt: prompt, Options: options, CorrectAnswer: answer, AcceptedAnswers: acceptedAnswers, Explanation: strings.TrimSpace(r.FormValue("explanation")), SortOrder: order})
	if err != nil {
		http.Error(w, "failed to create exercise", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10), http.StatusSeeOther)
}

func (h *CourseHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := h.theory.MarkTheoryRead(r.Context(), course.ID); err != nil {
		http.Error(w, "failed to save progress", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10)+"/practice", http.StatusSeeOther)
}

func (h *CourseHandler) Practice(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	exercises, err := h.theory.ListExercises(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load exercises", 500)
		return
	}
	practiceAction := "/courses/" + strconv.FormatInt(course.ID, 10) + "/practice"
	if rawBlockID := chi.URLParam(r, "block_id"); rawBlockID != "" {
		blockID, _ := strconv.ParseInt(rawBlockID, 10, 64)
		blocks, _ := h.theory.ListBlocks(r.Context(), course.ID)
		found := false
		for _, block := range blocks {
			if block.ID == blockID {
				found = true
				break
			}
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		filtered := make([]domain.TheoryExercise, 0)
		for _, exercise := range exercises {
			if exercise.TheoryBlockID != nil && *exercise.TheoryBlockID == blockID {
				filtered = append(filtered, exercise)
			}
		}
		exercises = filtered
		practiceAction = "/courses/" + strconv.FormatInt(course.ID, 10) + "/blocks/" + strconv.FormatInt(blockID, 10) + "/practice"
	}
	var result *domain.TheoryAnswerResult
	var answeredID int64
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		answeredID, _ = strconv.ParseInt(r.FormValue("exercise_id"), 10, 64)
		allowed := false
		for _, exercise := range exercises {
			if exercise.ID == answeredID {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "exercise does not belong to this practice", 400)
			return
		}
		value, submitErr := h.theory.SubmitAnswer(r.Context(), answeredID, r.FormValue("answer"))
		if submitErr != nil {
			http.Error(w, "failed to submit answer", 400)
			return
		}
		result = &value
	}
	progress, err := h.theory.Progress(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "failed to load progress", 500)
		return
	}
	data := pageData(r, map[string]any{"Title": "Practice · " + course.Name, "Course": activeCourse(h.tracks, r), "LearningCourse": course, "Exercises": exercises, "Progress": progress, "Result": result, "AnsweredID": answeredID, "PracticeAction": practiceAction})
	if err := h.practiceTemplates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render course practice", "error", err)
	}
}
