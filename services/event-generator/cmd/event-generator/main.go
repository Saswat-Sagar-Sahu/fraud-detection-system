package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/config"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/handler"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/kafka"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)

	cfg := config.Load()
	log.Printf(
		"configuration loaded http_addr=%s kafka_brokers=%s kafka_topic=%s",
		cfg.HTTPAddr,
		strings.Join(cfg.KafkaBrokers, ","),
		cfg.KafkaTopic,
	)

	publisher := kafka.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer publisher.Close()

	eventHandler := handler.NewEventHandler(publisher)
	mux := server.NewMux(eventHandler)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("event-generator listening on %s", cfg.HTTPAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
