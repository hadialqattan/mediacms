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
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(middleware.ErrorMiddleware)

	jwtManager := auth.NewJWTManager(jwtCfg)

	authHandler := handler.NewAuthHandler(svc, jwtManager)
	programHandler := handler.NewProgramHandler(svc)
	categoryHandler := handler.NewCategoryHandler(svc)
	importHandler := handler.NewImportHandler(svc)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/register", authHandler.CreateUser)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(jwtManager))

			r.Get("/programs", programHandler.List)
			r.Post("/programs", programHandler.Create)
			r.Get("/programs/{id}", programHandler.Get)
			r.Put("/programs/{id}", programHandler.Update)
			r.Post("/programs/{id}/publish", programHandler.Publish)
			r.Delete("/programs/{id}", programHandler.Delete)
			r.Put("/programs/{id}/categories", programHandler.AssignCategories)
			r.Get("/programs/{id}/categories", programHandler.GetCategories)

			r.Get("/categories", categoryHandler.List)
			r.Post("/categories", categoryHandler.Create)
			r.Get("/categories/{id}", categoryHandler.Get)

			r.Post("/import", importHandler.Import)
		})
	})

	return r
}
