package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/handler"
	"github.com/hadialqattan/mediacms/internal/cms/middleware"
	"github.com/hadialqattan/mediacms/internal/cms/service"
)

func NewRouter(svc *service.Service, jwtCfg config.JWTConfig) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // For now, for simplicity.
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(middleware.ErrorMiddleware)

	jwtManager := auth.NewJWTManager(jwtCfg)

	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(svc, jwtManager)
	userHandler := handler.NewUserHandler(svc)
	programHandler := handler.NewProgramHandler(svc)

	r.Get("/health", healthHandler.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(jwtManager))
			r.Post("/users", userHandler.CreateUser)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdminOrEditor(jwtManager))

			r.Get("/programs", programHandler.List)
			r.Post("/programs", programHandler.Create)
			r.Post("/programs/bulk", programHandler.BulkCreate)
			r.Delete("/programs/bulk", programHandler.BulkDelete)
			r.Get("/programs/{id}", programHandler.Get)
			r.Put("/programs/{id}", programHandler.Update)
			r.Post("/programs/{id}/publish", programHandler.Publish)
			r.Delete("/programs/{id}", programHandler.Delete)
		})
	})

	return r
}
