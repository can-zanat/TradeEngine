package main

import (
	"TradeEngine/internal"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const trashHoldTen = 10 * time.Second
const trashHoldFifteen = 15 * time.Second

func main() {
	registry := prometheus.NewRegistry()

	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	store := internal.NewStore()

	client := internal.NewClient()
	service := internal.NewService(*client, store)

	go func() {
		err := store.ListenRatesData(func(cValue float64, sValue string) {
			service.ManageRatesData(cValue, sValue)
		})
		if err != nil {
			log.Printf("Error occurs about MongoDB Change Stream: %v", err)
		}
	}()

	startServerWithGracefulShutdown(registry)
}

func startServerWithGracefulShutdown(registry *prometheus.Registry) {
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Kendi oluşturduğumuz registry üzerinden /metrics endpoint'ini sunuyoruz.
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:         ":9696",
		ReadTimeout:  trashHoldTen,
		WriteTimeout: trashHoldTen,
		IdleTimeout:  trashHoldFifteen,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("Starting metrics and health server on port 9696...")

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), trashHoldTen)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
