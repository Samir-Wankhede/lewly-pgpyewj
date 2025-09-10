package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	redisx "github.com/samirwankhede/lewly-pgpyewj/internal/redis"
	"github.com/samirwankhede/lewly-pgpyewj/internal/store"
)

type EventsService struct {
	log    *zap.Logger
	repo   *store.EventsRepository
	tokens *redisx.TokenBucket
}

func NewEventsService(log *zap.Logger, repo *store.EventsRepository, tokens *redisx.TokenBucket) *EventsService {
	return &EventsService{log: log, repo: repo, tokens: tokens}
}

func (s *EventsService) List(ctx context.Context, limit, offset int, q string, from, to *time.Time) ([]store.Event, error) {
	return s.repo.List(ctx, limit, offset, q, from, to)
}

func (s *EventsService) Get(ctx context.Context, id string) (*store.Event, int, error) {
	e, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	rem, _ := s.tokens.Remaining(ctx, id)
	return e, rem, nil
}
