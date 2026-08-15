package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"

	"github.com/go-chi/chi/v5"
)

type BoardsHandler struct {
	templates *template.Template
	boards    storage.BoardRepository
	courses   storage.LearningCourseRepository
	tracks    storage.CourseRepository
	logger    logger.Logger
	uploadDir string
}

func NewBoardsHandler(boards storage.BoardRepository, courses storage.LearningCourseRepository, tracks storage.CourseRepository, log logger.Logger) (*BoardsHandler, error) {
	tmpl, err := parsePage("board.html")
	return &BoardsHandler{templates: tmpl, boards: boards, courses: courses, tracks: tracks, logger: log, uploadDir: filepath.Join("data", "uploads")}, err
}

func (h *BoardsHandler) course(r *http.Request) (domain.LearningCourse, bool) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "course_id"), 10, 64)
	course, err := h.courses.Get(r.Context(), id)
	return course, err == nil && course.LanguageTrackID == activeCourse(h.tracks, r).ID
}
func (h *BoardsHandler) board(r *http.Request) (domain.Board, domain.LearningCourse, bool) {
	course, ok := h.course(r)
	if !ok {
		return domain.Board{}, course, false
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "board_id"), 10, 64)
	board, err := h.boards.Get(r.Context(), id)
	return board, course, err == nil && board.CourseID == course.ID
}

func (h *BoardsHandler) Create(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "Untitled board"
	}
	board, err := h.boards.Create(r.Context(), domain.Board{CourseID: course.ID, Name: name, Description: strings.TrimSpace(r.FormValue("description"))})
	if err != nil {
		http.Error(w, "failed to create board", 500)
		return
	}
	http.Redirect(w, r, "/courses/"+strconv.FormatInt(course.ID, 10)+"/boards/"+strconv.FormatInt(board.ID, 10), http.StatusSeeOther)
}

func (h *BoardsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := pageData(r, map[string]any{"Title": board.Name, "Course": activeCourse(h.tracks, r), "LearningCourse": course, "Board": board})
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("render board", "error", err)
	}
}

func (h *BoardsHandler) CreateText(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	kind := r.FormValue("kind")
	if kind != "note" {
		kind = "text"
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "content required", 400)
		return
	}
	_, err := h.boards.CreateNode(r.Context(), domain.BoardNode{BoardID: board.ID, Kind: kind, Title: strings.TrimSpace(r.FormValue("title")), Content: content, X: 100, Y: 100, Width: 280, Height: 170, Color: cleanBoardColor(r.FormValue("color"))})
	if err != nil {
		http.Error(w, "failed to create node", 500)
		return
	}
	http.Redirect(w, r, boardURL(course.ID, board.ID), http.StatusSeeOther)
}

func (h *BoardsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		http.Error(w, "file is too large", 400)
		return
	}
	file, header, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "media file required", 400)
		return
	}
	defer file.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	head = head[:n]
	mime := http.DetectContentType(head)
	kind, ext := "", ""
	switch mime {
	case "image/jpeg":
		kind, ext = "image", ".jpg"
	case "image/png":
		kind, ext = "image", ".png"
	case "image/webp":
		kind, ext = "image", ".webp"
	case "audio/mpeg":
		kind, ext = "audio", ".mp3"
	case "audio/ogg":
		kind, ext = "audio", ".ogg"
	case "audio/wav", "audio/x-wav":
		kind, ext = "audio", ".wav"
	default:
		http.Error(w, "unsupported media type", 400)
		return
	}
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		http.Error(w, "failed to prepare uploads", 500)
		return
	}
	name := randomFileName() + ext
	path := filepath.Join(h.uploadDir, name)
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "failed to store media", 500)
		return
	}
	_, copyErr := target.Write(head)
	if copyErr == nil {
		_, copyErr = io.Copy(target, file)
	}
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		http.Error(w, "failed to store media", 500)
		return
	}
	_, err = h.boards.CreateNode(r.Context(), domain.BoardNode{BoardID: board.ID, Kind: kind, Title: strings.TrimSpace(r.FormValue("title")), Content: header.Filename, MediaPath: "/uploads/" + name, X: 140, Y: 140, Width: kindWidth(kind), Height: 220, Color: "slate"})
	if err != nil {
		_ = os.Remove(path)
		http.Error(w, "failed to create media node", 500)
		return
	}
	http.Redirect(w, r, boardURL(course.ID, board.ID), http.StatusSeeOther)
}

func (h *BoardsHandler) Move(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "node_id"), 10, 64)
	var input struct{ X, Y float64 }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil {
		http.Error(w, "invalid position", 400)
		return
	}
	if input.X < -4000 || input.X > 12000 || input.Y < -4000 || input.Y > 12000 {
		http.Error(w, "position outside board", 400)
		return
	}
	if err := h.boards.MoveNode(r.Context(), board.ID, id, input.X, input.Y); err != nil {
		http.Error(w, "failed to move node", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *BoardsHandler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "node_id"), 10, 64)
	if err := h.boards.DeleteNode(r.Context(), board.ID, id); err != nil {
		http.Error(w, "failed to delete node", 500)
		return
	}
	http.Redirect(w, r, boardURL(course.ID, board.ID), http.StatusSeeOther)
}

func (h *BoardsHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	from, _ := strconv.ParseInt(r.FormValue("from_node_id"), 10, 64)
	to, _ := strconv.ParseInt(r.FormValue("to_node_id"), 10, 64)
	if from <= 0 || to <= 0 || from == to {
		http.Error(w, "choose two different cards", 400)
		return
	}
	if _, err := h.boards.CreateEdge(r.Context(), domain.BoardEdge{BoardID: board.ID, FromNodeID: from, ToNodeID: to, Label: strings.TrimSpace(r.FormValue("label")), Color: cleanBoardColor(r.FormValue("color"))}); err != nil {
		http.Error(w, "failed to connect cards", 400)
		return
	}
	http.Redirect(w, r, boardURL(course.ID, board.ID), http.StatusSeeOther)
}

func boardURL(courseID, boardID int64) string {
	return "/courses/" + strconv.FormatInt(courseID, 10) + "/boards/" + strconv.FormatInt(boardID, 10)
}
func cleanBoardColor(value string) string {
	switch value {
	case "amber", "mint", "rose", "slate":
		return value
	}
	return "violet"
}
func randomFileName() string {
	value := make([]byte, 16)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}
func kindWidth(kind string) float64 {
	if kind == "audio" {
		return 360
	}
	return 320
}
