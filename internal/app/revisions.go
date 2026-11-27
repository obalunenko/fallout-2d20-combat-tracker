package app

import (
	"context"
	"slices"

	"github.com/obalunenko/fallout/internal/domain"
)

// EntityKey identifies an entity a desktop editor can have open.
type EntityKey struct {
	Kind domain.AssistantEntityKind
	ID   string
}

// Revision returns the current revision of key. Revisions start at 1, so a
// command's ExpectedRevision of 0 means "do not check". Only assistant
// operations advance revisions (spec FR-022).
func (s *Service) Revision(key EntityKey) uint64 {
	s.revMu.Lock()
	defer s.revMu.Unlock()
	return s.revisions[key] + 1
}

// bumpRevisions advances keys when ctx carries the assistant source.
// Callers hold the write gate.
func (s *Service) bumpRevisions(ctx context.Context, keys ...EntityKey) {
	if SourceFromContext(ctx) != domain.SourceAssistant {
		return
	}
	s.revMu.Lock()
	defer s.revMu.Unlock()
	for _, k := range keys {
		if k.ID != "" {
			s.revisions[k]++
		}
	}
}

// tombstone marks key as deleted when ctx carries the assistant source.
func (s *Service) tombstone(ctx context.Context, key EntityKey) {
	if SourceFromContext(ctx) != domain.SourceAssistant || key.ID == "" {
		return
	}
	s.revMu.Lock()
	defer s.revMu.Unlock()
	s.deleted[key] = true
	s.revisions[key]++
}

// checkRevision rejects a desktop save whose editor opened at an older
// revision. expected 0 skips the check. Callers hold the write gate.
func (s *Service) checkRevision(key EntityKey, expected uint64) error {
	if expected == 0 {
		return nil
	}
	s.revMu.Lock()
	defer s.revMu.Unlock()
	if s.deleted[key] {
		return domain.ErrEntityDeleted
	}
	if s.revisions[key]+1 != expected {
		return domain.ErrStaleEdit
	}
	return nil
}

func campaignKey(id string) EntityKey  { return EntityKey{Kind: domain.EntityCampaign, ID: id} }
func encounterKey(id string) EntityKey { return EntityKey{Kind: domain.EntityEncounter, ID: id} }

// encounterChangeKeys returns the revision keys an encounter save touches: the
// encounter, plus its campaign when party combatants are present, because the
// save writes linked character stats back to the player characters.
func encounterChangeKeys(enc *domain.Encounter) []EntityKey {
	if enc == nil {
		return nil
	}
	keys := []EntityKey{encounterKey(enc.ID)}
	if hasPartyCombatant(enc) {
		keys = append(keys, campaignKey(enc.CampaignID))
	}
	return keys
}

// encounterEvent builds the change event for an encounter save.
func encounterEvent(enc *domain.Encounter) ChangeEvent {
	ev := ChangeEvent{Kinds: []domain.AssistantEntityKind{domain.EntityEncounter}}
	if enc == nil {
		return ev
	}
	ev.EncounterID = enc.ID
	ev.CampaignID = enc.CampaignID
	if hasPartyCombatant(enc) {
		ev.Kinds = append(ev.Kinds, domain.EntityCampaign)
	}
	return ev
}

// hasPartyCombatant reports whether enc has a party combatant, whose save
// writes linked character stats back to the campaign.
func hasPartyCombatant(enc *domain.Encounter) bool {
	return slices.ContainsFunc(enc.Combatants, func(c domain.Combatant) bool { return c.Side == domain.SideParty })
}
