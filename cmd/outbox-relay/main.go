package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/outboxrelay"
	"github.com/hadialqattan/mediacms/internal/outboxrelay/repository"
	"github.com/hadialqattan/mediacms/internal/shared/postgres"
)

func main() {
	cfg := config.Load()

	pool, err := postgres.NewConnectionPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	outboxRepo := repository.NewOutboxRepo(pool)
	queue := repository.NewQueue(cfg.RedisAddr)

	relay := outboxrelay.NewRelay(outboxRepo, queue, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting Outbox Relay...")
		relay.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down relay...")
	cancel()
}
