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
	style := r.FormValue("style")
	kind := "text"
	if style == "note" {
		kind = "note"
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "content required", 400)
		return
	}
	x, y := nextBoardPosition(len(board.Nodes))
	color, width, height := cleanBoardColor(r.FormValue("color")), 280.0, 170.0
	if style == "label" {
		kind, color, width, height = "text", "clear", 380, 110
	}
	_, err := h.boards.CreateNode(r.Context(), domain.BoardNode{BoardID: board.ID, Kind: kind, Title: strings.TrimSpace(r.FormValue("title")), Content: content, X: x, Y: y, Width: width, Height: height, Color: color, TextColor: cleanTextColor(r.FormValue("text_color"))})
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
	x, y := nextBoardPosition(len(board.Nodes))
	_, err = h.boards.CreateNode(r.Context(), domain.BoardNode{BoardID: board.ID, Kind: kind, Title: strings.TrimSpace(r.FormValue("title")), Content: header.Filename, MediaPath: "/uploads/" + name, X: x, Y: y, Width: kindWidth(kind), Height: 220, Color: "slate", TextColor: "white"})
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

func (h *BoardsHandler) Resize(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "node_id"), 10, 64)
	var input struct{ Width, Height float64 }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil || input.Width < 180 || input.Width > 800 || input.Height < 100 || input.Height > 700 {
		http.Error(w, "invalid card size", 400)
		return
	}
	if err := h.boards.ResizeNode(r.Context(), board.ID, id, input.Width, input.Height); err != nil {
		http.Error(w, "failed to resize node", 400)
		return
	}
	writeBoardOK(w)
}

func (h *BoardsHandler) Edit(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "node_id"), 10, 64)
	var input struct{ Title, Content, Color, TextColor string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid card", 400)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		http.Error(w, "content required", 400)
		return
	}
	if err := h.boards.UpdateNode(r.Context(), domain.BoardNode{ID: id, BoardID: board.ID, Title: input.Title, Content: input.Content, Color: cleanBoardColor(input.Color), TextColor: cleanTextColor(input.TextColor)}); err != nil {
		http.Error(w, "failed to edit node", 400)
		return
	}
	writeBoardOK(w)
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

func (h *BoardsHandler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	board, course, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "edge_id"), 10, 64)
	if err := h.boards.DeleteEdge(r.Context(), board.ID, id); err != nil {
		http.Error(w, "failed to delete connection", 500)
		return
	}
	http.Redirect(w, r, boardURL(course.ID, board.ID), http.StatusSeeOther)
}

func (h *BoardsHandler) CreateStroke(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var input struct {
		Kind   string  `json:"kind"`
		Color  string  `json:"color"`
		Width  float64 `json:"width"`
		Points []struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"points"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid drawing", 400)
		return
	}
	if input.Kind != "pen" && input.Kind != "arrow" || len(input.Points) < 2 || len(input.Points) > 2000 || input.Width < 1 || input.Width > 16 {
		http.Error(w, "invalid drawing", 400)
		return
	}
	for _, point := range input.Points {
		if point.X < -4000 || point.X > 12000 || point.Y < -4000 || point.Y > 12000 {
			http.Error(w, "drawing outside board", 400)
			return
		}
	}
	points, _ := json.Marshal(input.Points)
	stroke, err := h.boards.CreateStroke(r.Context(), domain.BoardStroke{BoardID: board.ID, Kind: input.Kind, Points: string(points), Color: cleanStrokeColor(input.Color), Width: input.Width})
	if err != nil {
		http.Error(w, "failed to save drawing", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": stroke.ID})
}

func (h *BoardsHandler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var input struct {
		Content, TextColor string
		X, Y               float64
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&input); err != nil {
		http.Error(w, "invalid text", 400)
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" || input.X < -4000 || input.X > 12000 || input.Y < -4000 || input.Y > 12000 {
		http.Error(w, "invalid text", 400)
		return
	}
	node, err := h.boards.CreateNode(r.Context(), domain.BoardNode{BoardID: board.ID, Kind: "text", Content: input.Content, X: input.X, Y: input.Y, Width: 380, Height: 110, Color: "clear", TextColor: cleanTextColor(input.TextColor)})
	if err != nil {
		http.Error(w, "failed to create text", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": node.ID})
}

func (h *BoardsHandler) DeleteStroke(w http.ResponseWriter, r *http.Request) {
	board, _, ok := h.board(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "stroke_id"), 10, 64)
	if err := h.boards.DeleteStroke(r.Context(), board.ID, id); err != nil {
		http.Error(w, "failed to erase drawing", 500)
		return
	}
	writeBoardOK(w)
}

func boardURL(courseID, boardID int64) string {
	return "/courses/" + strconv.FormatInt(courseID, 10) + "/boards/" + strconv.FormatInt(boardID, 10)
}
func cleanBoardColor(value string) string {
	switch value {
	case "amber", "mint", "rose", "slate", "clear":
		return value
	}
	return "violet"
}
func cleanTextColor(value string) string {
	switch value {
	case "white", "amber", "mint", "rose", "violet":
		return value
	}
	return "white"
}
func cleanStrokeColor(value string) string {
	switch value {
	case "amber", "violet", "mint", "rose", "white", "marker_amber", "marker_violet", "marker_mint", "marker_rose", "marker_white":
		return value
	}
	return "amber"
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
func nextBoardPosition(count int) (float64, float64) {
	return 360 + float64(count%3)*320, 120 + float64(count/3)*220
}
func writeBoardOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}
