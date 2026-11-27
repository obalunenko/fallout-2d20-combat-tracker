package app

import (
	"time"

	"github.com/obalunenko/fallout/internal/domain"
)

type CreateCampaignCommand struct {
	ID        string
	Name      string
	StartDate time.Time
	Players   []domain.NewCampaignPlayer
}

type UpdateCampaignCommand struct {
	CampaignID string
	Name       string
	StartDate  time.Time
	Players    []domain.NewCampaignPlayer
	// ExpectedRevision is the campaign revision the editor opened at
	// (Service.Revision). 0 skips the stale-edit check.
	ExpectedRevision uint64
}

type CreateEncounterCommand struct {
	ID         string
	Name       string
	Combatants []domain.Combatant
}

type UpdateEncounterCommand struct {
	EncounterID string
	Name        string
	Combatants  []domain.Combatant
	// ExpectedRevision is the encounter revision the editor opened at
	// (Service.Revision). 0 skips the stale-edit check.
	ExpectedRevision uint64
}

type ApplyDamageCommand struct {
	CombatantID string
	DamageType  domain.DamageType
	Location    domain.BodyLocation
	Amount      int
}

type HealCommand struct {
	CombatantID string
	Amount      int
}

type SaveMonsterTemplatesCommand struct {
	Monsters []domain.Combatant
}
