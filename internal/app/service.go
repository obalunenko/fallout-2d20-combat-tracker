package app

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type Service struct {
	repo EncounterRepository
	logf func(string, ...any)

	sideEffectFailuresMu sync.Mutex
	sideEffectFailures   map[string]uint64
	operationTimeout     time.Duration

	// gate serializes writers and lets readers share (see gate.go).
	gate *semaphore.Weighted

	subsMu    sync.Mutex
	subs      map[uint64]func(ChangeEvent)
	nextSubID uint64

	revMu     sync.Mutex
	revisions map[EntityKey]uint64
	deleted   map[EntityKey]bool
}

type sideEffectCategory string

const (
	sideEffectCategoryAudit         sideEffectCategory = "audit"
	sideEffectCategoryTelemetry     sideEffectCategory = "telemetry"
	sideEffectCategoryNotifications sideEffectCategory = "notifications"

	sideEffectNameAppendEncounterLog      = "append_encounter_log"
	sideEffectNameAppendAssistantActivity = "append_assistant_activity"
	defaultOperationTimeout               = 5 * time.Second
)

func NewService(repo EncounterRepository) *Service {
	return NewServiceWithLogfAndTimeout(repo, log.Printf, defaultOperationTimeout)
}

func NewServiceWithLogf(repo EncounterRepository, logf func(string, ...any)) *Service {
	return NewServiceWithLogfAndTimeout(repo, logf, defaultOperationTimeout)
}

func NewServiceWithLogfAndTimeout(repo EncounterRepository, logf func(string, ...any), operationTimeout time.Duration) *Service {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Service{
		repo:               repo,
		logf:               logf,
		sideEffectFailures: make(map[string]uint64),
		operationTimeout:   operationTimeout,
		gate:               semaphore.NewWeighted(gateWeight),
		subs:               make(map[uint64]func(ChangeEvent)),
		revisions:          make(map[EntityKey]uint64),
		deleted:            make(map[EntityKey]bool),
	}
}

func (s *Service) contextForOperation(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.operationTimeout <= 0 {
		return ctx, func() {}
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.operationTimeout)
}
