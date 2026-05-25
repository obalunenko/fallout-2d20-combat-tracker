package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) CreateCampaign(ctx context.Context, id, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return s.ExecuteCreateCampaign(ctx, CreateCampaignCommand{
		ID:        id,
		Name:      name,
		StartDate: startDate,
		Players:   players,
	})
}

func (s *Service) ExecuteCreateCampaign(ctx context.Context, cmd CreateCampaignCommand) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	if strings.TrimSpace(cmd.ID) == "" {
		cmd.ID = uuid.NewString()
	}
	if err := prepareCampaignPlayers(cmd.Players); err != nil {
		return nil, err
	}
	return s.repo.CreateCampaign(ctx, cmd.ID, cmd.Name, cmd.StartDate, cmd.Players)
}

func (s *Service) GetActiveCampaign(ctx context.Context) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.GetActiveCampaign(ctx)
}

func (s *Service) ListCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.ListCampaigns(ctx)
}

func (s *Service) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return s.repo.ListCampaignPlayers(ctx, campaignID)
}

func (s *Service) ActivateCampaign(ctx context.Context, campaignID string) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if err := s.repo.ActivateCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	return s.repo.GetActiveCampaign(ctx)
}

func (s *Service) UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return s.ExecuteUpdateCampaign(ctx, UpdateCampaignCommand{
		CampaignID: campaignID,
		Name:       name,
		StartDate:  startDate,
		Players:    players,
	})
}

func (s *Service) ExecuteUpdateCampaign(ctx context.Context, cmd UpdateCampaignCommand) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.CampaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	if err := prepareCampaignPlayers(cmd.Players); err != nil {
		return nil, err
	}
	return s.repo.UpdateCampaign(ctx, cmd.CampaignID, cmd.Name, cmd.StartDate, cmd.Players)
}

func prepareCampaignPlayers(players []domain.NewCampaignPlayer) error {
	for i := range players {
		if err := prepareCampaignPlayer(&players[i]); err != nil {
			return err
		}
	}
	return nil
}

func prepareCampaignPlayer(player *domain.NewCampaignPlayer) error {
	if strings.TrimSpace(player.PlayerName) == "" {
		return fmt.Errorf("player name is required")
	}
	if strings.TrimSpace(player.Character.Name) == "" {
		return fmt.Errorf("character name is required for player %q", player.PlayerName)
	}
	player.Character.Name = strings.TrimSpace(player.Character.Name)
	if err := domain.ValidateCombatant(player.Character, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("player %q", player.PlayerName),
		RequireName: true,
		MinLevel:    1,
	}); err != nil {
		return err
	}
	if strings.TrimSpace(player.Character.ID) == "" {
		player.Character.ID = uuid.NewString()
	}
	domain.NormalizeCombatantHP(&player.Character)
	player.Character.Side = domain.SideParty
	player.Character.XP = 0
	return nil
}
