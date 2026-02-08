package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/hadialqattan/mediacms/docs/cms-api"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/repository"
	"github.com/hadialqattan/mediacms/internal/cms/router"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/postgres"
)

// @title           MediaCMS API
// @version         0.0.1
// @description     Content Management System API for MediaCMS

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg := config.LoadCMS()

	pool, err := postgres.NewConnectionPool(context.Background(), cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:            cfg.Redis.Addr,
		MaxRetries:      cfg.Redis.MaxRetries,
		MinRetryBackoff: cfg.Redis.MinRetryBackoff,
		MaxRetryBackoff: cfg.Redis.MaxRetryBackoff,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	programRepo := repository.NewProgramRepo(pool)
	outboxRepo := repository.NewOutboxRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	sessionRepo := repository.NewSessionRepo(redisClient, cfg.JWT)
	jwtManager := auth.NewJWTManager(cfg.JWT)

	svc := service.NewService(programRepo, outboxRepo, userRepo, sessionRepo, jwtManager, pool)

	if err := svc.SeedDefaultAdmin(context.Background(), cfg.DefaultAdmin.Email, cfg.DefaultAdmin.Password); err != nil {
		log.Fatalf("Failed to seed default admin: %v", err)
	}

	r := router.NewRouter(svc, cfg.JWT)

	// Note: This is not supposed to be here (part of router).
	//		 But for the sake of time, I'll leave it here ;-).
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Starting CMS API on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
}
