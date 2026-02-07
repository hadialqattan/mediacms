package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/repository"
	"github.com/hadialqattan/mediacms/internal/cms/router"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/postgres"
)

func main() {
	cfg := config.Load()

	pool, err := postgres.NewConnectionPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:            cfg.RedisAddr,
		MaxRetries:      3,
		MinRetryBackoff: 500 * time.Millisecond,
		MaxRetryBackoff: time.Second,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	programRepo := repository.NewProgramRepo(pool)
	categoryRepo := repository.NewCategoryRepo(pool)
	sourceRepo := repository.NewSourceRepo(pool)
	outboxRepo := repository.NewOutboxRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	sessionRepo := repository.NewSessionRepo(redisClient, cfg.JWT)
	jwtManager := auth.NewJWTManager(cfg.JWT)

	svc := service.NewService(programRepo, categoryRepo, sourceRepo, outboxRepo, userRepo, sessionRepo, jwtManager, pool)

	if err := svc.SeedDefaultAdmin(context.Background(), cfg.DefaultAdmin.Email, cfg.DefaultAdmin.Password); err != nil {
		log.Fatalf("Failed to seed default admin: %v", err)
	}

	r := router.NewRouter(svc, cfg.JWT)

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
