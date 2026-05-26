package domain

import "time"

type Campaign struct {
	ID        string
	Name      string
	StartDate time.Time
	Resources Resources
	UpdatedAt time.Time
}

type NewCampaignPlayer struct {
	PlayerName string
	Character  Combatant
	Inactive   bool
}

type CampaignCharacter struct {
	PlayerID   string
	PlayerName string
	Character  Combatant
	Active     bool
}
