package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)

	// Web routes
	mux.Get("/", app.dashboardHandler)

	// API routes
	mux.Group(func(r chi.Router) {
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/healthcheck", app.healthcheckHandler)
			r.Get("/websites", app.listWebsitesHandler)
			r.Get("/websites/{id}", app.getWebsiteHandler)
		})
	})

	return mux
}
