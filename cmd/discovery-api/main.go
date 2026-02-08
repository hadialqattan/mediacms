package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/hadialqattan/mediacms/docs/discovery-api"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/typesense/typesense-go/typesense"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/discovery"
	"github.com/hadialqattan/mediacms/internal/discovery/repository"
	"github.com/hadialqattan/mediacms/internal/discovery/router"
)

// @title           MediaCMS Discovery API
// @version         0.0.1
// @description     Public search and discovery API for MediaCMS programs

// @host      localhost:8081
// @BasePath  /

func main() {
	cfg := config.LoadDiscovery()

	typesenseClient := typesense.NewClient(
		typesense.WithServer(cfg.TypesenseAddress),
		typesense.WithAPIKey(cfg.TypesenseAPIKey),
	)

	searchIndex := repository.NewSearchIndex(typesenseClient)
	svc := discovery.NewService(searchIndex)

	r := router.NewRouter(svc)

	// Note: This is not supposed to be here (part of router).
	//		 But for the sake of time, I'll leave it here ;-).
	r.Get("/swagger/*", httpSwagger.WrapHandler)

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
