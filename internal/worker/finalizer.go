package worker

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type FinalizePayload struct {
	Type           string  `json:"type"`
	BookingID      string  `json:"booking_id"`
	EventID        string  `json:"event_id"`
	UserID         string  `json:"user_id"`
	IdempotencyKey *string `json:"idempotency_key"`
}

type Finalizer struct {
	log *zap.Logger
	db  *store.DB
	c   *kafkax.Consumer
	dlq *kafkax.Producer
}

func NewFinalizer(log *zap.Logger, db *store.DB, c *kafkax.Consumer, dlq *kafkax.Producer) *Finalizer {
	return &Finalizer{log: log, db: db, c: c, dlq: dlq}
}

func (f *Finalizer) Run(ctx context.Context) error {
	for {
		m, err := f.c.Fetch(ctx)
		if err != nil {
			return err
		}

		var p FinalizePayload
		if err := json.Unmarshal(m.Value, &p); err != nil {
			_ = f.dlq.Publish(ctx, m.Key, m.Value)
			_ = f.c.Commit(ctx, m)
			continue
		}

		// Basic retry loop
		var attempt int
		for {
			attempt++
			err = f.db.FinalizeBookingTx(ctx, p.BookingID, p.EventID)
			if err == nil || err == store.ErrSoldOut {
				break
			}
			if attempt >= 3 {
				_ = f.dlq.Publish(ctx, m.Key, m.Value)
				break
			}
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
		}

		// Note: future enhancement: on cancellation events, promote earliest waitlist user.
		_ = f.c.Commit(ctx, m)
	}
}
