-- +migrate Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    email TEXT UNIQUE,
    phone TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    venue TEXT,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    capacity INT NOT NULL,
    reserved INT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_name_trgm ON events USING gin(name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS event_capacity (
    event_id UUID PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    capacity INT NOT NULL,
    reserved_count INT NOT NULL DEFAULT 0,
    held_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    status TEXT CHECK (status IN ('pending','booked','cancelled','waitlisted','expired')),
    seats JSONB NULL,
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    version INT DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_bookings_event_status ON bookings(event_id, status);
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_idem ON bookings(idempotency_key);

CREATE TABLE IF NOT EXISTS waitlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    position INT NOT NULL,
    opted_out BOOL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_waitlist_event_position ON waitlist(event_id, position);

CREATE TABLE IF NOT EXISTS seats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    seat_label TEXT,
    status TEXT CHECK (status IN ('available','held','booked')),
    held_until TIMESTAMPTZ NULL,
    held_by_booking UUID NULL
);

CREATE INDEX IF NOT EXISTS idx_seats_event_label ON seats(event_id, seat_label);

CREATE TABLE IF NOT EXISTS analytics_aggregates (
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    date DATE,
    total_bookings INT DEFAULT 0,
    cancellations INT DEFAULT 0,
    capacity_utilization NUMERIC,
    PRIMARY KEY (event_id, date)
);

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity TEXT,
    action TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);


