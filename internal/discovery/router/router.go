package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/hadialqattan/mediacms/internal/discovery"
	"github.com/hadialqattan/mediacms/internal/discovery/handler"
)

func NewRouter(service *discovery.Service) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	programHandler := handler.NewProgramHandler(service)
	healthHandler := handler.NewHealthHandler()

	r.Get("/health", healthHandler.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/programs", func(r chi.Router) {
			r.Get("/", programHandler.Search)
			r.Get("/{id}", programHandler.Get)
			r.Get("/recent", programHandler.GetRecent)
			r.Get("/facets", programHandler.GetFacets)
		})
	})

	return r
}
