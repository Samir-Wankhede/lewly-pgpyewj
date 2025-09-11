package main

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/samirwankhede/lewly-pgpyewj/internal/config"
	"github.com/samirwankhede/lewly-pgpyewj/internal/logger"
	"github.com/samirwankhede/lewly-pgpyewj/internal/metrics"
	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log := logger.New(cfg.Env)
	ctx := context.Background()

	db, err := store.NewDB(ctx, cfg.PostgresURL, int32(cfg.MaxDBConnections))
	if err != nil {
		log.Fatal("db", zap.Error(err))
	}
	defer db.Close()
	tokens := redisx.NewTokenBucket(cfg.RedisAddr)

	// Simple reconciliation: compare events.capacity - events.reserved vs Redis tokens
	metrics.ReconciliationRunsTotal.Inc()
	rows, err := db.Pool.Query(ctx, `SELECT id, capacity, reserved FROM events`)
	if err != nil {
		log.Fatal("query", zap.Error(err))
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var capacity, reserved int
		if err := rows.Scan(&id, &capacity, &reserved); err != nil {
			log.Error("scan", zap.Error(err))
			continue
		}
		desired := capacity - reserved
		rem, _ := tokens.Remaining(ctx, id)
		if rem != desired {
			diff := desired - rem
			if diff > 0 {
				_ = tokens.Release(ctx, id, diff)
			} else if diff < 0 {
				// consume extra tokens
				for i := 0; i < -diff; i++ {
					_, _ = tokens.Reserve(ctx, id, 1)
				}
			}
			metrics.ReconciliationFixesTotal.Inc()
			log.Info("reconciled", zap.String("event", id), zap.Int("desired", desired), zap.Int("was", rem))
		}
	}
	fmt.Println("reconciliation complete at", time.Now())
}
