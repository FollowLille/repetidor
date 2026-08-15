package web

import (
	"net/http"
	"repetidor/internal/logger"
	"repetidor/internal/web/handlers"
	webmiddleware "repetidor/internal/web/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter(container *handlers.Container, appLogger logger.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(webmiddleware.RequestLogger(appLogger))

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./data/uploads"))))

	r.Get("/", container.Home.ServeHTTP)
	r.Get("/arena", container.Arena.ServeHTTP)
	r.Method(http.MethodGet, "/settings", container.Settings)
	r.Method(http.MethodPost, "/settings", container.Settings)
	r.Get("/locale", container.Settings.SetLocale)
	r.Post("/course", container.Settings.SetCourse)
	r.Post("/track", container.Settings.SetCourse)
	r.Method(http.MethodGet, "/import", container.Import)
	r.Method(http.MethodPost, "/import", container.Import)
	r.Method(http.MethodGet, "/courses", container.Courses)
	r.Method(http.MethodPost, "/courses", container.Courses)
	r.Get("/courses/{course_id}", container.Course.ServeHTTP)
	r.Post("/courses/{course_id}/edit", container.Course.Update)
	r.Post("/courses/{course_id}/delete", container.Course.Delete)
	r.Post("/courses/{course_id}/blocks", container.Course.CreateBlock)
	r.Post("/courses/{course_id}/blocks/{block_id}/delete", container.Course.DeleteBlock)
	r.Post("/courses/{course_id}/exercises", container.Course.CreateExercise)
	r.Post("/courses/{course_id}/exercises/{exercise_id}/delete", container.Course.DeleteExercise)
	r.Post("/courses/{course_id}/boards", container.Boards.Create)
	r.Get("/courses/{course_id}/boards/{board_id}", container.Boards.ServeHTTP)
	r.Post("/courses/{course_id}/boards/{board_id}/nodes", container.Boards.CreateText)
	r.Post("/courses/{course_id}/boards/{board_id}/media", container.Boards.Upload)
	r.Post("/courses/{course_id}/boards/{board_id}/nodes/{node_id}/move", container.Boards.Move)
	r.Post("/courses/{course_id}/boards/{board_id}/nodes/{node_id}/resize", container.Boards.Resize)
	r.Post("/courses/{course_id}/boards/{board_id}/nodes/{node_id}/edit", container.Boards.Edit)
	r.Post("/courses/{course_id}/boards/{board_id}/nodes/{node_id}/delete", container.Boards.DeleteNode)
	r.Post("/courses/{course_id}/boards/{board_id}/edges", container.Boards.CreateEdge)
	r.Post("/courses/{course_id}/boards/{board_id}/edges/{edge_id}/delete", container.Boards.DeleteEdge)
	r.Post("/courses/{course_id}/read", container.Course.MarkRead)
	r.Get("/courses/{course_id}/practice", container.Course.Practice)
	r.Post("/courses/{course_id}/practice", container.Course.Practice)
	r.Get("/stats", container.Stats.ServeHTTP)
	r.Get("/stats/sessions/{session_id}", container.Session.ServeHTTP)
	r.Post("/stats/sessions/{session_id}/abandon", container.Session.ServeHTTP)
	r.Method(http.MethodGet, "/train/{train_mode}", container.Training)
	r.Method(http.MethodPost, "/train/{train_mode}", container.Training)
	r.Method(http.MethodGet, "/topics", container.Topics)
	r.Method(http.MethodPost, "/topics", container.Topics)
	r.Get("/topics/{topic_name}", container.Topic.ServeHTTP)
	r.Post("/topics/{topic_name}/words", container.Topic.CreateWord)
	r.Post("/topics/{topic_name}/words/{word_id}/delete", container.Topic.DeleteWord)
	r.Method(http.MethodGet, "/topics/{topic_name}/words/{word_id}/edit", container.WordEdit)
	r.Method(http.MethodPost, "/topics/{topic_name}/words/{word_id}/edit", container.WordEdit)
	r.Method(http.MethodGet, "/topics/{topic_name}/edit", container.TopicEdit)
	r.Method(http.MethodPost, "/topics/{topic_name}/edit", container.TopicEdit)
	r.Post("/topics/{topic_name}/delete", container.TopicEdit.Delete)

	return r
}
