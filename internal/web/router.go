package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"repetidor/internal/web/handlers"
)

func NewRouter() (http.Handler, error) {
	r := chi.NewRouter()

	homeHandler, err := handlers.NewHomeHandler()
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", homeHandler.ServeHTTP)

	return r, nil
}
