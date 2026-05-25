package app

import (
	"context"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
)

type EncounterReader interface {
	Get(ctx context.Context) (*domain.Encounter, error)
	List(ctx context.Context) ([]domain.EncounterSummary, error)
	GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error)
	ListPartyMembers(ctx context.Context) ([]domain.Combatant, error)
}

type EncounterWriter interface {
	Save(ctx context.Context, encounter *domain.Encounter) error
	UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error)
	Activate(ctx context.Context, encounterID string) error
	SoftDelete(ctx context.Context, encounterID string) error
}

type MonsterTemplateRepository interface {
	ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error)
	UpsertMonsterTemplate(ctx context.Context, monster domain.Combatant) (domain.Combatant, error)
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	GetActiveCampaign(ctx context.Context) (*domain.Campaign, error)
	ListCampaigns(ctx context.Context) ([]domain.Campaign, error)
	ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error)
	ActivateCampaign(ctx context.Context, campaignID string) error
}

type EncounterLogRepository interface {
	AppendEncounterLog(ctx context.Context, encounterID string, round int, message string) error
	ListEncounterLogs(ctx context.Context, encounterID string) ([]domain.EncounterLog, error)
}

type EncounterRepository interface {
	EncounterReader
	EncounterWriter
	MonsterTemplateRepository
	CampaignRepository
	EncounterLogRepository
}
