package store

import (
	"context"
)

type WaitlistRepository struct{ db *DB }

func NewWaitlistRepository(db *DB) *WaitlistRepository { return &WaitlistRepository{db: db} }

func (r *WaitlistRepository) Add(ctx context.Context, eventID, userID string) (int, error) {
	var pos int
	err := r.db.Pool.QueryRow(ctx, `WITH next AS (
        SELECT COALESCE(MAX(position),0)+1 AS p FROM waitlist WHERE event_id=$1
    ) INSERT INTO waitlist (event_id, user_id, position) VALUES ($1,$2,(SELECT p FROM next))
    RETURNING position`, eventID, userID).Scan(&pos)
	return pos, err
}

func (r *WaitlistRepository) OptOut(ctx context.Context, eventID, userID string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE waitlist SET opted_out=TRUE WHERE event_id=$1 AND user_id=$2`, eventID, userID)
	return err
}

// NextActive returns the next non-opted-out waitlist user for an event.
func (r *WaitlistRepository) NextActive(ctx context.Context, eventID string) (string, string, int, error) {
	var id, userID string
	var position int
	err := r.db.Pool.QueryRow(ctx, `SELECT id, user_id, position FROM waitlist WHERE event_id=$1 AND opted_out=FALSE ORDER BY position ASC LIMIT 1`, eventID).
		Scan(&id, &userID, &position)
	if err != nil {
		return "", "", 0, err
	}
	return id, userID, position, nil
}

// Remove deletes a waitlist row by id
func (r *WaitlistRepository) Remove(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM waitlist WHERE id=$1`, id)
	return err
}
