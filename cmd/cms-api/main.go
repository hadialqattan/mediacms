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

	"thmanyah.com/content-platform/config"
	"thmanyah.com/content-platform/internal/cms/auth"
	"thmanyah.com/content-platform/internal/cms/repository"
	"thmanyah.com/content-platform/internal/cms/router"
	"thmanyah.com/content-platform/internal/cms/service"
	"thmanyah.com/content-platform/internal/shared/postgres"
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

	svc := service.NewService(programRepo, categoryRepo, sourceRepo, outboxRepo, userRepo, sessionRepo, jwtManager)

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
