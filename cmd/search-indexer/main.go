package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/typesense/typesense-go/typesense"

	"thmanyah.com/content-platform/config"
	"thmanyah.com/content-platform/internal/discovery/repository"
	"thmanyah.com/content-platform/internal/searchindexer"
	searchindexerrepo "thmanyah.com/content-platform/internal/searchindexer/repository"
	"thmanyah.com/content-platform/internal/shared/postgres"
)

func main() {
	cfg := config.Load()

	pool, err := postgres.NewConnectionPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	typesenseClient := typesense.NewClient(
		typesense.WithServer(cfg.TypesenseAddress),
		typesense.WithAPIKey(cfg.TypesenseAPIKey),
	)

	programReader := searchindexerrepo.NewProgramReader(pool)
	searchIndex := repository.NewSearchIndex(typesenseClient)
	
	worker := searchindexer.NewWorker(programReader, searchIndex)
	server, mux := searchindexer.NewWorkerAndMux(worker)

	go func() {
		log.Println("Starting Search Indexer worker...")
		if err := server.Run(mux); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	server.Shutdown()
}
