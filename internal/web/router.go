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

	r.Get("/", container.Home.ServeHTTP)
	r.Get("/arena", container.Arena.ServeHTTP)
	r.Method(http.MethodGet, "/settings", container.Settings)
	r.Method(http.MethodPost, "/settings", container.Settings)
	r.Get("/locale", container.Settings.SetLocale)
	r.Post("/course", container.Settings.SetCourse)
	r.Method(http.MethodGet, "/import", container.Import)
	r.Method(http.MethodPost, "/import", container.Import)
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
