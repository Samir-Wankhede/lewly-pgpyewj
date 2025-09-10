-- +migrate Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS analytics_aggregates;
DROP TABLE IF EXISTS seats;
DROP INDEX IF EXISTS idx_waitlist_event_position;
DROP TABLE IF EXISTS waitlist;
DROP INDEX IF EXISTS idx_bookings_user;
DROP INDEX IF EXISTS idx_bookings_event_status;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS event_capacity;
DROP INDEX IF EXISTS idx_events_name_trgm;
DROP INDEX IF EXISTS idx_events_start_time;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS users;

