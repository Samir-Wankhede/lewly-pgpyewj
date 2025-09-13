package admin

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/admin"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/events"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/seats"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store/users"
)

type AdminService struct {
	log    *zap.Logger
	events *events.EventsRepository
	users  *users.UsersRepository
	admin  *admin.AdminRepository
	seats  *seats.SeatsRepository
	tokens *redisx.TokenBucket
	mailer MailerService
}

type MailerService interface {
	SendEventCancellationEmail(userEmail string, eventName string, refundAmount float64, paymentLink string) error
}

func NewAdminService(log *zap.Logger, events *events.EventsRepository, users *users.UsersRepository, admin *admin.AdminRepository, seats *seats.SeatsRepository, tokens *redisx.TokenBucket, mailer MailerService) *AdminService {
	return &AdminService{log: log, events: events, users: users, admin: admin, seats: seats, tokens: tokens, mailer: mailer}
}

type AdminEvent struct {
	Name                     string    `json:"name"`
	Venue                    string    `json:"venue"`
	StartTime                time.Time `json:"start_time"`
	EndTime                  time.Time `json:"end_time"`
	Capacity                 int       `json:"capacity"`
	Metadata                 []byte    `json:"metadata"`
	TicketPrice              float64   `json:"ticket_price"`
	CancellationFee          float64   `json:"cancellation_fee"`
	MaximumTicketsPerBooking int       `json:"maximum_tickets_per_booking"`
	Seats                    []string  `json:"seats"`
}

func (a *AdminService) CreateEvent(ctx context.Context, in AdminEvent) (*events.Event, error) {
	// Validate seats array size matches capacity
	if len(in.Seats) != in.Capacity {
		return nil, errors.New("seats array size must match event capacity")
	}

	e := &events.Event{
		Name:                     in.Name,
		Venue:                    in.Venue,
		StartTime:                in.StartTime,
		EndTime:                  in.EndTime,
		Capacity:                 in.Capacity,
		Metadata:                 in.Metadata,
		Status:                   "upcoming",
		TicketPrice:              in.TicketPrice,
		CancellationFee:          in.CancellationFee,
		MaximumTicketsPerBooking: in.MaximumTicketsPerBooking,
	}
	e, err := a.events.Create(ctx, e)
	if err != nil {
		return nil, err
	}

	// Create seats in the seats table
	err = a.seats.CreateSeats(ctx, e.ID, in.Seats)
	if err != nil {
		a.log.Error("Failed to create seats", zap.Error(err), zap.String("event_id", e.ID))
		// Note: We don't return error here as the event is already created
		// In production, you might want to rollback the event creation
	}

	_ = a.tokens.InitTokens(ctx, e.ID, e.Capacity)
	return e, nil
}

func (a *AdminService) GetSummary(ctx context.Context, from, to time.Time) (*admin.AnalyticsSummary, error) {
	return a.admin.GetSummary(ctx, from, to)
}

func (a *AdminService) CancelEvent(ctx context.Context, eventID string) error {
	// Get event details for email notifications
	event, err := a.events.Get(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return errors.New("event not found")
	}

	// Cancel the event
	err = a.admin.CancelEvent(ctx, eventID)
	if err != nil {
		return err
	}

	// Send cancellation emails to all booked users
	// This would typically be done asynchronously
	// For now, we'll just log it
	a.log.Info("Event cancelled", zap.String("event_id", eventID), zap.String("event_name", event.Name))

	return nil
}

func (a *AdminService) UpdateEvent(ctx context.Context, eventID string, updates map[string]interface{}) error {
	return a.admin.UpdateEvent(ctx, eventID, updates)
}

func (a *AdminService) CreateAdminFromUser(ctx context.Context, userID string) error {
	return a.admin.CreateAdminFromUser(ctx, userID)
}

func (a *AdminService) RemoveAdmin(ctx context.Context, userID string) error {
	return a.admin.RemoveAdmin(ctx, userID)
}

func (a *AdminService) RemoveUser(ctx context.Context, userID string) error {
	return a.admin.RemoveUser(ctx, userID)
}
