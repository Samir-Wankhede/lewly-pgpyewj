package store

import (
	"context"
	"time"
)

type BookingStatus string

const (
	BookingPending    BookingStatus = "pending"
	BookingBooked     BookingStatus = "booked"
	BookingCancelled  BookingStatus = "cancelled"
	BookingWaitlisted BookingStatus = "waitlisted"
	BookingExpired    BookingStatus = "expired"
)

type Booking struct {
	ID             string
	UserID         string
	EventID        string
	Status         BookingStatus
	Seats          []byte
	IdempotencyKey *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
}

type BookingsRepository struct{ db *DB }

func NewBookingsRepository(db *DB) *BookingsRepository { return &BookingsRepository{db: db} }

func (r *BookingsRepository) CreatePending(ctx context.Context, userID, eventID string, idem *string) (*Booking, error) {
	b := Booking{UserID: userID, EventID: eventID, Status: BookingPending, IdempotencyKey: idem}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO bookings (user_id, event_id, status, idempotency_key) VALUES ($1,$2,'pending',$3)
         RETURNING id, created_at, updated_at, version`, userID, eventID, idem,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.Version)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BookingsRepository) GetByIdempotency(ctx context.Context, idem string) (*Booking, error) {
	var b Booking
	var idemPtr *string
	err := r.db.Pool.QueryRow(ctx, `SELECT id, user_id, event_id, status, seats, idempotency_key, created_at, updated_at, version FROM bookings WHERE idempotency_key=$1`, idem).
		Scan(&b.ID, &b.UserID, &b.EventID, &b.Status, &b.Seats, &idemPtr, &b.CreatedAt, &b.UpdatedAt, &b.Version)
	if err != nil {
		return nil, err
	}
	b.IdempotencyKey = idemPtr
	return &b, nil
}

// CancelBookingTx sets booking to cancelled and decrements events.reserved if it was booked.
func (r *BookingsRepository) CancelBookingTx(ctx context.Context, bookingID string) (*Booking, bool, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var b Booking
	var idem *string
	if err := tx.QueryRow(ctx, `SELECT id, user_id, event_id, status, seats, idempotency_key, created_at, updated_at, version FROM bookings WHERE id=$1 FOR UPDATE`, bookingID).
		Scan(&b.ID, &b.UserID, &b.EventID, &b.Status, &b.Seats, &idem, &b.CreatedAt, &b.UpdatedAt, &b.Version); err != nil {
		return nil, false, err
	}
	b.IdempotencyKey = idem
	if b.Status == BookingCancelled {
		return &b, false, tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `UPDATE bookings SET status='cancelled', updated_at=now() WHERE id=$1`, bookingID); err != nil {
		return nil, false, err
	}
	wasBooked := false
	if b.Status == BookingBooked {
		wasBooked = true
		if _, err := tx.Exec(ctx, `UPDATE events SET reserved = GREATEST(reserved-1,0), updated_at=now() WHERE id=$1`, b.EventID); err != nil {
			return nil, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	b.Status = BookingCancelled
	return &b, wasBooked, nil
}

func (r *BookingsRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Booking, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id, user_id, event_id, status, seats, idempotency_key, created_at, updated_at, version FROM bookings WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Booking
	for rows.Next() {
		var b Booking
		var idem *string
		if err := rows.Scan(&b.ID, &b.UserID, &b.EventID, &b.Status, &b.Seats, &idem, &b.CreatedAt, &b.UpdatedAt, &b.Version); err != nil {
			return nil, err
		}
		b.IdempotencyKey = idem
		out = append(out, b)
	}
	return out, rows.Err()
}
