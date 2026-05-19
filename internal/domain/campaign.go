package domain

type Campaign struct {
	ID        string
	Name      string
	StartDate string
	UpdatedAt string
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
