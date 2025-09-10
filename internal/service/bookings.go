package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"

	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type BookingsService struct {
	log    *zap.Logger
	repo   *store.BookingsRepository
	tokens *redisx.TokenBucket
	prod   *kafkax.Producer
	wait   *store.WaitlistRepository
}

type BookingRequest struct {
	UserID         string   `json:"user_id"`
	Seats          []string `json:"seats"`
	IdempotencyKey *string  `json:"idempotency_key"`
}

type BookingResponse struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
	Position  int    `json:"position,omitempty"`
}

func NewBookingsService(log *zap.Logger, repo *store.BookingsRepository, tokens *redisx.TokenBucket, prod *kafkax.Producer, wait *store.WaitlistRepository) *BookingsService {
	return &BookingsService{log: log, repo: repo, tokens: tokens, prod: prod, wait: wait}
}

func (s *BookingsService) Create(ctx context.Context, eventID string, req BookingRequest) (*BookingResponse, int, error) {
	// Idempotency check
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		if b, err := s.repo.GetByIdempotency(ctx, *req.IdempotencyKey); err == nil && b != nil {
			return &BookingResponse{BookingID: b.ID, Status: string(b.Status)}, 200, nil
		}
	}

	ok, err := s.tokens.Reserve(ctx, eventID, 1)
	if err != nil {
		return nil, 500, err
	}
	if ok {
		b, err := s.repo.CreatePending(ctx, req.UserID, eventID, req.IdempotencyKey)
		if err != nil {
			return nil, 500, err
		}
		_ = s.tokens.SetHold(ctx, eventID, b.ID, 3*time.Minute)
		payload := map[string]any{
			"type":            "finalize_booking",
			"booking_id":      b.ID,
			"event_id":        eventID,
			"user_id":         req.UserID,
			"idempotency_key": req.IdempotencyKey,
		}
		by, _ := json.Marshal(payload)
		if err := s.prod.Publish(ctx, []byte(eventID), by); err != nil {
			s.log.Error("kafka publish error", zap.Error(err))
		}
		return &BookingResponse{BookingID: b.ID, Status: "pending"}, 202, nil
	}

	// Fallback: Auto waitlist (simplified placeholder; full impl later)
	return &BookingResponse{Status: "waitlisted", Position: 0}, 200, nil
}

var ErrValidation = errors.New("validation error")

func (s *BookingsService) Cancel(ctx context.Context, bookingID string) (map[string]any, int, error) {
	b, wasBooked, err := s.repo.CancelBookingTx(ctx, bookingID)
	if err != nil {
		return nil, 409, err
	}
	// release a token when a booked reservation is cancelled
	if wasBooked {
		_ = s.tokens.Release(ctx, b.EventID, 1)
		if s.wait != nil {
			if id, userID, _, err := s.wait.NextActive(ctx, b.EventID); err == nil && userID != "" {
				if pb, cerr := s.repo.CreatePending(ctx, userID, b.EventID, nil); cerr == nil {
					payload := map[string]any{
						"type":            "finalize_booking",
						"booking_id":      pb.ID,
						"event_id":        b.EventID,
						"user_id":         userID,
						"idempotency_key": nil,
					}
					by, _ := json.Marshal(payload)
					_ = s.prod.Publish(ctx, []byte(b.EventID), by)
					_ = s.wait.Remove(ctx, id)
				}
			}
		}
	}
	return map[string]any{"booking_id": b.ID, "status": b.Status}, 200, nil
}
