package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"MetalTracker/middleman/internal/api"
	"MetalTracker/middleman/internal/poller"
	"MetalTracker/middleman/internal/service"
	"MetalTracker/middleman/internal/store"
	"MetalTracker/middleman/internal/upstream"
)

func main() {
	apiKey := os.Getenv("METALPRICE_API_KEY")
	if apiKey == "" {
		log.Fatal("METALPRICE_API_KEY is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "prices.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil && filepath.Dir(dbPath) != "." {
		log.Fatalf("create db dir: %v", err)
	}

	priceStore, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer priceStore.Close()

	interval := time.Hour
	if raw := os.Getenv("POLL_INTERVAL"); raw != "" {
		parsed, parseErr := time.ParseDuration(raw)
		if parseErr != nil {
			log.Fatalf("POLL_INTERVAL: %v", parseErr)
		}
		interval = parsed
	}

	svc := service.New(priceStore, upstream.NewClient(apiKey))
	server := api.New(svc, os.Getenv("MIDDLEMAN_API_KEY"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go poller.RunHourly(ctx, svc, interval)

	httpServer := &http.Server{
		Addr:              api.ListenAddrFromEnv(),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("middleman listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Printf("middleman stopped")
}
