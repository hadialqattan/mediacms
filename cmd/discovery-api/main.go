package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/typesense/typesense-go/typesense"

	"thmanyah.com/content-platform/config"
	"thmanyah.com/content-platform/internal/discovery"
	"thmanyah.com/content-platform/internal/discovery/repository"
	"thmanyah.com/content-platform/internal/discovery/router"
)

func main() {
	cfg := config.Load()

	typesenseClient := typesense.NewClient(
		typesense.WithServer(cfg.TypesenseAddress),
		typesense.WithAPIKey(cfg.TypesenseAPIKey),
	)

	searchIndex := repository.NewSearchIndex(typesenseClient)
	svc := discovery.NewService(searchIndex)

	r := router.NewRouter(svc)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Discovery API on port %s", cfg.Port)
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
