package worker

import (
	"context"
	"encoding/json"
	"time"

	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type FinalizePayload struct {
	Type           string  `json:"type"`
	BookingID      string  `json:"booking_id"`
	EventID        string  `json:"event_id"`
	UserID         string  `json:"user_id"`
	IdempotencyKey *string `json:"idempotency_key"`
}

type Finalizer struct {
	log        *zap.Logger
	db         *store.DB
	c          *kafkax.Consumer
	dlq        *kafkax.Producer
	maxWorkers int
}

func NewFinalizer(log *zap.Logger, db *store.DB, c *kafkax.Consumer, dlq *kafkax.Producer, maxWorkers int) *Finalizer {
	return &Finalizer{log: log, db: db, c: c, dlq: dlq, maxWorkers: maxWorkers}
}

func (f *Finalizer) Run(ctx context.Context) error {
	workerCount := f.maxWorkers
	sem := make(chan struct{}, workerCount) // concurrency limit

	for {
		m, err := f.c.Fetch(ctx)
		if err != nil {
			return err
		}

		sem <- struct{}{} // acquire slot
		go func(m kafka.Message) {
			defer func() { <-sem }() // release slot

			if err := f.handleMessage(ctx, m); err != nil {
				f.log.Error("failed to handle", zap.Error(err))
			}

			_ = f.c.Commit(ctx, m)
		}(m)
	}
}

func (f *Finalizer) handleMessage(ctx context.Context, m kafka.Message) error {
	var p FinalizePayload
	if err := json.Unmarshal(m.Value, &p); err != nil {
		// JSON parse failure is not retriable → straight to DLQ
		_ = f.dlq.Publish(ctx, m.Key, m.Value)
		return err
	}

	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = f.db.FinalizeBookingTx(ctx, p.BookingID, p.EventID)
		if err == nil {
			return nil // success
		}

		f.log.Warn("finalize booking failed, will retry",
			zap.Int("attempt", attempt),
			zap.Error(err),
			zap.String("booking_id", p.BookingID),
			zap.String("event_id", p.EventID),
		)

		// exponential backoff: 100ms, 200ms, 400ms
		time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
	}

	// after 3 attempts, push to DLQ
	_ = f.dlq.Publish(ctx, m.Key, m.Value)
	return err
}
