package redisx

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type TimeoutBucket struct {
	client *redis.Client
}

func NewTimeoutBucket(addr string) *TimeoutBucket {
	c := redis.NewClient(&redis.Options{Addr: addr})
	return &TimeoutBucket{client: c}
}

func (t *TimeoutBucket) NilError() error {
	return redis.Nil
}

func (t *TimeoutBucket) AddBooking(ctx context.Context, eventID string, bookingID string) error {
	key := eventID + ":" + bookingID
	return t.client.Set(ctx, key, 1, 15*time.Minute).Err()
}

func (t *TimeoutBucket) Close() { _ = t.client.Close() }
