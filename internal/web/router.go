package web

import (
	"net/http"
	"repetidor/internal/web/handlers"

	"github.com/go-chi/chi/v5"
)

func NewRouter(container *handlers.Container) http.Handler {
	r := chi.NewRouter()

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", container.Home.ServeHTTP)
	r.Get("/train/{train_mode}", container.Training.ServeHTTP)
	r.Method(http.MethodGet, "/topics", container.Topics)
	r.Method(http.MethodPost, "/topics", container.Topics)
	r.Get("/topics/{topic_name}", container.Topic.ServeHTTP)

	return r
}
