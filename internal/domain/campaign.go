package domain

import "time"

type Campaign struct {
	ID        string
	Name      string
	StartDate time.Time
	UpdatedAt time.Time
}

type NewCampaignPlayer struct {
	PlayerName string
	Character  Combatant
}

type CampaignCharacter struct {
	PlayerID   string
	PlayerName string
	Character  Combatant
	Active     bool
}
