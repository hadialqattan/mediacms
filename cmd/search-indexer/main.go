package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/typesense/typesense-go/typesense"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/discovery/repository"
	"github.com/hadialqattan/mediacms/internal/searchindexer"
)

func main() {
	cfg := config.Load()

	typesenseClient := typesense.NewClient(
		typesense.WithServer(cfg.TypesenseAddress),
		typesense.WithAPIKey(cfg.TypesenseAPIKey),
	)

	searchIndex := repository.NewSearchIndex(typesenseClient)

	worker := searchindexer.NewWorker(searchIndex)
	server, mux := searchindexer.NewWorkerAndMux(worker, cfg.RedisAddr)

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
