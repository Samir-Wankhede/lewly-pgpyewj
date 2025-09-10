package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type AdminService struct {
	log    *zap.Logger
	events *store.EventsRepository
	tokens *redisx.TokenBucket
}

func NewAdminService(log *zap.Logger, events *store.EventsRepository, tokens *redisx.TokenBucket) *AdminService {
	return &AdminService{log: log, events: events, tokens: tokens}
}

type AdminEvent struct {
	Name      string    `json:"name"`
	Venue     string    `json:"venue"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Capacity  int       `json:"capacity"`
	Metadata  []byte    `json:"metadata"`
}

func (a *AdminService) CreateEvent(ctx context.Context, in AdminEvent) (*store.Event, error) {
	e := &store.Event{Name: in.Name, Venue: in.Venue, StartTime: in.StartTime, EndTime: in.EndTime, Capacity: in.Capacity, Metadata: in.Metadata}
	e, err := a.events.Create(ctx, e)
	if err != nil {
		return nil, err
	}
	_ = a.tokens.InitTokens(ctx, e.ID, e.Capacity)
	return e, nil
}
