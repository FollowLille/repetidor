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
	r.Get("/train/{train_mode}", container.Training.ServeHTTP)
	r.Method(http.MethodGet, "/topics", container.Topics)
	r.Method(http.MethodPost, "/topics", container.Topics)
	r.Get("/topics/{topic_name}", container.Topic.ServeHTTP)
	r.Method(http.MethodGet, "/topics/{topic_name}/edit", container.TopicEdit)
	r.Method(http.MethodPost, "/topics/{topic_name}/edit", container.TopicEdit)

	return r
}
