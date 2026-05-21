package fyneui

import "github.com/obalunenko/fallout/internal/domain"

type uiState struct {
	enc                 *domain.Encounter
	activeCampaign      *domain.Campaign
	selectedIndex       int
	expandedCombatantID string
}
