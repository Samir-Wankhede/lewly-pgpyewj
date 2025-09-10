package store

import (
	"context"
	"fmt"
	"time"
)

type Event struct {
	ID        string
	Name      string
	Venue     string
	StartTime time.Time
	EndTime   time.Time
	Capacity  int
	Reserved  int
	Metadata  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EventsRepository struct{ db *DB }

func NewEventsRepository(db *DB) *EventsRepository { return &EventsRepository{db: db} }

func (r *EventsRepository) List(ctx context.Context, limit, offset int, q string, from, to *time.Time) ([]Event, error) {
	base := `SELECT id, name, venue, start_time, end_time, capacity, reserved, metadata, created_at, updated_at FROM events WHERE 1=1`
	args := []any{}
	idx := func() int { return len(args) + 1 }
	if q != "" {
		base += fmt.Sprintf(" AND name ILIKE '%%' || $%d || '%%'", idx())
		args = append(args, q)
	}
	if from != nil {
		base += fmt.Sprintf(" AND start_time >= $%d", idx())
		args = append(args, *from)
	}
	if to != nil {
		base += fmt.Sprintf(" AND end_time <= $%d", idx())
		args = append(args, *to)
	}
	base += fmt.Sprintf(" ORDER BY start_time ASC LIMIT $%d OFFSET $%d", idx(), idx()+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Name, &e.Venue, &e.StartTime, &e.EndTime, &e.Capacity, &e.Reserved, &e.Metadata, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EventsRepository) Get(ctx context.Context, id string) (*Event, error) {
	var e Event
	err := r.db.Pool.QueryRow(ctx, `SELECT id, name, venue, start_time, end_time, capacity, reserved, metadata, created_at, updated_at FROM events WHERE id=$1`, id).
		Scan(&e.ID, &e.Name, &e.Venue, &e.StartTime, &e.EndTime, &e.Capacity, &e.Reserved, &e.Metadata, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventsRepository) Create(ctx context.Context, e *Event) (*Event, error) {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO events (name, venue, start_time, end_time, capacity, reserved, metadata) VALUES ($1,$2,$3,$4,$5,0,$6)
         RETURNING id, created_at, updated_at`,
		e.Name, e.Venue, e.StartTime, e.EndTime, e.Capacity, e.Metadata,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// no itoa helper required
