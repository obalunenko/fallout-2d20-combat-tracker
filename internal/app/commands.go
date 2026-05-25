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
