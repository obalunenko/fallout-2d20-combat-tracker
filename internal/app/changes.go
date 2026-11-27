package app

import "github.com/obalunenko/fallout/internal/domain"

// ChangeEvent describes one successful mutating operation.
type ChangeEvent struct {
	Source      domain.OperationSource
	Kinds       []domain.AssistantEntityKind
	CampaignID  string
	EncounterID string
}

// SubscribeChanges registers fn to receive a ChangeEvent after every
// successful mutating operation. fn runs synchronously on the caller's
// goroutine after the operation has released the gate, so it must hand work
// off quickly (for example with fyne.Do). The returned func unsubscribes.
func (s *Service) SubscribeChanges(fn func(ChangeEvent)) (unsubscribe func()) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	s.nextSubID++
	id := s.nextSubID
	s.subs[id] = fn
	return func() {
		s.subsMu.Lock()
		defer s.subsMu.Unlock()
		delete(s.subs, id)
	}
}

func (s *Service) publish(ev ChangeEvent) {
	s.subsMu.Lock()
	fns := make([]func(ChangeEvent), 0, len(s.subs))
	for _, fn := range s.subs {
		fns = append(fns, fn)
	}
	s.subsMu.Unlock()
	for _, fn := range fns {
		fn(ev)
	}
}
