package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/typesense/typesense-go/typesense"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/discovery/repository"
	"github.com/hadialqattan/mediacms/internal/searchindexer"
)

func main() {
	cfg := config.LoadSearchIndexer()

	typesenseClient := typesense.NewClient(
		typesense.WithServer(cfg.TypesenseAddress),
		typesense.WithAPIKey(cfg.TypesenseAPIKey),
	)

	searchIndex := repository.NewSearchIndex(typesenseClient)

	log.Println("Ensuring Typesense collection exists...")
	if err := createCollectionWithRetry(searchIndex.(*repository.SearchIndex)); err != nil {
		log.Fatalf("Failed to create Typesense collection after retries: %v", err)
	}
	log.Println("Typesense collection ready")

	worker := searchindexer.NewWorker(searchIndex)
	server, mux := searchindexer.NewWorkerAndMux(worker, cfg.Redis.Addr)

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

// Note: This is not supposed to be here (part of router), but for the sake of time, I'll leave it here ;-).
func createCollectionWithRetry(searchIndex *repository.SearchIndex) error {
	const maxRetries = 10
	const baseDelay = 2 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := searchIndex.CreateCollectionIfNotExists(context.Background())
		if err == nil {
			return nil
		}

		lastErr = err
		delay := baseDelay * time.Duration(1<<attempt)
		log.Printf("Attempt %d/%d failed: %v. Retrying in %v...", attempt+1, maxRetries, err, delay)

		time.Sleep(delay)
	}

	return lastErr
}
