package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/samirwankhede/lewly-pgpyewj/internal/config"
	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	"github.com/samirwankhede/lewly-pgpyewj/internal/logger"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
	"github.com/samirwankhede/lewly-pgpyewj/internal/worker"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log := logger.New(cfg.Env)
	log.Info("worker starting")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := store.NewDB(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer db.Close()

	consumer := kafkax.NewConsumer([]string{cfg.KafkaBrokers}, "evently-finalizer", "bookings")
	defer consumer.Close()
	dlq := kafkax.NewProducer([]string{cfg.KafkaBrokers}, "bookings-dlq")
	defer dlq.Close()

	f := worker.NewFinalizer(log, db, consumer, dlq)
	go func() { _ = f.Run(ctx) }()

	<-ctx.Done()
	log.Info("worker stopped")
}
