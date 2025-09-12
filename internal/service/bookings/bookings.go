package bookings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kafkax "github.com/samirwankhede/lewly-pgpyewj/internal/kafka"
	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/bookings"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/events"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/waitlist"
)

type BookingsService struct {
	log    *zap.Logger
	repo   *bookings.BookingsRepository
	events *events.EventsRepository
	tokens *redisx.TokenBucket
	prod   *kafkax.Producer
	wait   *waitlist.WaitlistRepository
	mailer MailerService
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

type MailerService interface {
	SendCancellationEmail(userEmail string, cancellationFee float64, paymentLink string) error
	SendWaitlistPromotionEmail(userEmail string, eventName string, paymentLink string) error
}

func NewBookingsService(log *zap.Logger, repo *bookings.BookingsRepository, events *events.EventsRepository, tokens *redisx.TokenBucket, prod *kafkax.Producer, wait *waitlist.WaitlistRepository, mailer MailerService) *BookingsService {
	return &BookingsService{log: log, repo: repo, events: events, tokens: tokens, prod: prod, wait: wait, mailer: mailer}
}

func (s *BookingsService) Create(ctx context.Context, eventID string, req BookingRequest) (*BookingResponse, int, error) {
	// Check if event exists and is not expired
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return nil, 500, err
	}
	if event == nil {
		return nil, 404, errors.New("event not found")
	}

	// Check if event is expired
	if event.EndTime.Before(time.Now()) {
		// Update event status to expired
		s.events.UpdateStatus(ctx, eventID, "expired")
		return nil, 400, errors.New("event is expired")
	}

	// Check if user is trying to book more than maximum allowed
	if len(req.Seats) > event.MaximumTicketsPerBooking {
		return nil, 400, fmt.Errorf("cannot book more than %d tickets", event.MaximumTicketsPerBooking)
	}

	// Idempotency check
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		if b, err := s.repo.GetByIdempotency(ctx, *req.IdempotencyKey); err == nil && b != nil {
			return &BookingResponse{BookingID: b.ID, Status: b.Status}, 200, nil
		}
	}

	// Reserve tokens for the number of seats requested
	ok, err := s.tokens.Reserve(ctx, eventID, len(req.Seats))
	if err != nil {
		return nil, 500, err
	}

	if ok {
		b, err := s.repo.CreatePending(ctx, req.UserID, eventID, req.IdempotencyKey)
		if err != nil {
			return nil, 500, err
		}

		// Store seats in booking
		seatsJSON, _ := json.Marshal(req.Seats)
		err = s.repo.UpdateSeats(ctx, b.ID, seatsJSON)
		if err != nil {
			s.log.Error("failed to update seats", zap.Error(err))
		}

		_ = s.tokens.SetHold(ctx, eventID, b.ID, 3*time.Minute)
		payload := map[string]any{
			"type":            "finalize_booking",
			"booking_id":      b.ID,
			"event_id":        eventID,
			"user_id":         req.UserID,
			"seats":           req.Seats,
			"idempotency_key": req.IdempotencyKey,
		}
		by, _ := json.Marshal(payload)
		if err := s.prod.Publish(ctx, []byte(eventID), by); err != nil {
			s.log.Error("kafka publish error", zap.Error(err))
		}
		return &BookingResponse{BookingID: b.ID, Status: "pending"}, 202, nil
	}

	// Fallback: Auto waitlist
	position, err := s.wait.Add(ctx, eventID, req.UserID)
	if err != nil {
		return nil, 500, err
	}

	return &BookingResponse{Status: "waitlisted", Position: position}, 200, nil
}

var ErrValidation = errors.New("validation error")

func (s *BookingsService) Cancel(ctx context.Context, bookingID string) (map[string]any, int, error) {
	b, wasBooked, err := s.repo.CancelBookingTx(ctx, bookingID)
	if err != nil {
		return nil, 409, err
	}

	// release tokens when a booked reservation is cancelled
	if wasBooked {
		// Get the number of seats from the booking
		var seats []string
		if len(b.Seats) > 0 {
			json.Unmarshal(b.Seats, &seats)
		}
		seatCount := len(seats)
		if seatCount == 0 {
			seatCount = 1 // fallback
		}

		_ = s.tokens.Release(ctx, b.EventID, seatCount)

		// Send cancellation email with fee and payment link
		if s.mailer != nil {
			// Get user email (you'd need to implement this)
			// For now, we'll skip the email sending
			// s.mailer.SendCancellationEmail(userEmail, event.CancellationFee, paymentLink)
		}

		// Promote next person from waitlist
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

					// Send waitlist promotion email
					if s.mailer != nil {
						// s.mailer.SendWaitlistPromotionEmail(userEmail, eventName, paymentLink)
					}
				}
			}
		}
	}
	return map[string]any{"booking_id": b.ID, "status": b.Status}, 200, nil
}

func (s *BookingsService) GetBookingStatus(ctx context.Context, bookingID string) (string, error) {
	return s.repo.GetBookingStatus(ctx, bookingID)
}

func (s *BookingsService) GetAvailableSeats(ctx context.Context, eventID string) ([]string, error) {
	return s.events.GetAvailableSeats(ctx, eventID)
}

func (s *BookingsService) ListUserBookings(ctx context.Context, userID string, limit, offset int) ([]*bookings.Booking, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

func (s *BookingsService) FinalizeBooking(ctx context.Context, bookingID string, seats []string, amountPaid float64) error {
	seatsJSON, _ := json.Marshal(seats)
	return s.repo.FinalizeBooking(ctx, bookingID, seatsJSON, amountPaid)
}
