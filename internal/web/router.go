package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"repetidor/internal/web/handlers"
)

func NewRouter(container *handlers.Container) http.Handler {
	r := chi.NewRouter()

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		container.Home.ServeHTTP(w, r)
	})

	r.Get("/train/{train_mode}", func(w http.ResponseWriter, r *http.Request) {
		container.Training.ServeHTTP(w, r)
	})

	r.Get("/topics", func(w http.ResponseWriter, r *http.Request) {
		container.Topics.ServeHTTP(w, r)
	})

	r.Post("/topics", func(w http.ResponseWriter, r *http.Request) {
		container.Topics.ServeHTTP(w, r)
	})

	r.Get("/topics/{topic_name}", func(w http.ResponseWriter, r *http.Request) {
		container.Topic.ServeHTTP(w, r)
	})

	return r
}
