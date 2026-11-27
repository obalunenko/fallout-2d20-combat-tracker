package app

import (
	"context"
	"errors"

	"github.com/obalunenko/fallout/internal/domain"
)

// gateWeight is the weight a writer takes. A writer takes all of it, so it
// runs alone; a reader takes 1, so readers share the gate.
const gateWeight = 1 << 20

// operation is one gated Service call. Every exported Service method starts
// one with beginRead or beginWrite and defers end. Unexported helpers never
// start one, so a gated call never re-enters the gate.
type operation struct {
	s      *Service
	weight int64
	cancel context.CancelFunc
	events []ChangeEvent
}

func (s *Service) beginRead(ctx context.Context) (context.Context, *operation, error) {
	return s.begin(ctx, 1)
}

func (s *Service) beginWrite(ctx context.Context) (context.Context, *operation, error) {
	return s.begin(ctx, gateWeight)
}

func (s *Service) begin(ctx context.Context, weight int64) (context.Context, *operation, error) {
	ctx, cancel := s.contextForOperation(ctx)
	if err := s.gate.Acquire(ctx, weight); err != nil {
		cancel()
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil, domain.ErrTrackerBusy
		}
		return nil, nil, err
	}
	return ctx, &operation{s: s, weight: weight, cancel: cancel}, nil
}

// changed queues a change event. Events are published by end, after the gate
// is released, so a subscriber may call back into the Service.
func (op *operation) changed(ctx context.Context, ev ChangeEvent) {
	ev.Source = SourceFromContext(ctx)
	op.events = append(op.events, ev)
}

func (op *operation) end() {
	op.s.gate.Release(op.weight)
	op.cancel()
	for _, ev := range op.events {
		op.s.publish(ev)
	}
}
