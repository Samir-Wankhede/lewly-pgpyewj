package store

import (
	"context"
	"errors"
)

var ErrSoldOut = errors.New("event sold out")

// FinalizeBookingTx finalizes a booking by locking the event and updating counts atomically.
func (d *DB) FinalizeBookingTx(ctx context.Context, bookingID, eventID string) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var capacity, reserved int
	if err := tx.QueryRow(ctx, `SELECT capacity, reserved FROM events WHERE id=$1 FOR UPDATE`, eventID).Scan(&capacity, &reserved); err != nil {
		return err
	}
	if reserved >= capacity {
		// mark booking waitlisted if not already
		if _, err := tx.Exec(ctx, `UPDATE bookings SET status='waitlisted', updated_at=now() WHERE id=$1 AND status='pending'`, bookingID); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return ErrSoldOut
	}

	// set booked and increment reserved idempotently
	if _, err := tx.Exec(ctx, `UPDATE bookings SET status='booked', updated_at=now() WHERE id=$1 AND status='pending'`, bookingID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE events SET reserved = reserved + 1, updated_at=now() WHERE id=$1`, eventID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
