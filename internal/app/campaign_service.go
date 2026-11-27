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
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, invalidf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, invalidf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, invalidf("add at least one player")
	}
	if strings.TrimSpace(cmd.ID) == "" {
		cmd.ID = uuid.NewString()
	}
	if err := prepareCampaignPlayers(cmd.Players); err != nil {
		return nil, err
	}
	campaign, err := s.repo.CreateCampaign(ctx, cmd.ID, cmd.Name, cmd.StartDate, cmd.Players)
	if err != nil {
		return nil, err
	}
	s.bumpRevisions(ctx, campaignKey(campaign.ID))
	op.changed(ctx, ChangeEvent{Kinds: []domain.AssistantEntityKind{domain.EntityCampaign}, CampaignID: campaign.ID})
	return campaign, nil
}

func (s *Service) GetActiveCampaign(ctx context.Context) (*domain.Campaign, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.GetActiveCampaign(ctx)
}

func (s *Service) ListCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.ListCampaigns(ctx)
}

func (s *Service) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if strings.TrimSpace(campaignID) == "" {
		return nil, invalidf("campaign id is required")
	}
	return s.repo.ListCampaignPlayers(ctx, campaignID)
}

func (s *Service) ActivateCampaign(ctx context.Context, campaignID string) (*domain.Campaign, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if strings.TrimSpace(campaignID) == "" {
		return nil, invalidf("campaign id is required")
	}
	if err := s.repo.ActivateCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	campaign, err := s.repo.GetActiveCampaign(ctx)
	if err != nil {
		return nil, err
	}
	op.changed(ctx, ChangeEvent{Kinds: []domain.AssistantEntityKind{domain.EntityCampaign, domain.EntityEncounter}, CampaignID: campaign.ID})
	return campaign, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return s.ExecuteUpdateCampaign(ctx, UpdateCampaignCommand{
		CampaignID: campaignID,
		Name:       name,
		StartDate:  startDate,
		Players:    players,
	})
}

// ExecuteUpdateCampaign stores the campaign's details and full roster. When
// cmd.ExpectedRevision is set and the assistant changed the campaign after
// that revision, it saves nothing and returns domain.ErrStaleEdit.
func (s *Service) ExecuteUpdateCampaign(ctx context.Context, cmd UpdateCampaignCommand) (*domain.Campaign, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if err := s.checkRevision(campaignKey(cmd.CampaignID), cmd.ExpectedRevision); err != nil {
		return nil, err
	}
	return s.updateCampaign(ctx, op, cmd)
}

// updateCampaign runs a campaign update. The caller holds the write gate.
func (s *Service) updateCampaign(ctx context.Context, op *operation, cmd UpdateCampaignCommand) (*domain.Campaign, error) {
	if strings.TrimSpace(cmd.CampaignID) == "" {
		return nil, invalidf("campaign id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, invalidf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, invalidf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, invalidf("add at least one player")
	}
	if err := prepareCampaignPlayers(cmd.Players); err != nil {
		return nil, err
	}
	campaign, err := s.repo.UpdateCampaign(ctx, cmd.CampaignID, cmd.Name, cmd.StartDate, cmd.Players)
	if err != nil {
		return nil, err
	}
	keys := []EntityKey{campaignKey(cmd.CampaignID)}
	if SourceFromContext(ctx) == domain.SourceAssistant {
		// The update may rewrite this campaign's encounters (linked character
		// sync and removal of unavailable characters).
		if summaries, listErr := s.repo.List(ctx); listErr == nil {
			for _, sum := range summaries {
				if sum.CampaignID == cmd.CampaignID {
					keys = append(keys, encounterKey(sum.ID))
				}
			}
		}
	}
	s.bumpRevisions(ctx, keys...)
	op.changed(ctx, ChangeEvent{
		Kinds:      []domain.AssistantEntityKind{domain.EntityCampaign, domain.EntityPlayerCharacter, domain.EntityEncounter},
		CampaignID: cmd.CampaignID,
	})
	return campaign, nil
}

// prepareCampaignPlayers validates and normalizes a roster. Player names are
// unique in a campaign, ignoring case and surrounding spaces in any script.
func prepareCampaignPlayers(players []domain.NewCampaignPlayer) error {
	seen := make(map[string]bool, len(players))
	for i := range players {
		if err := prepareCampaignPlayer(&players[i]); err != nil {
			return err
		}
		key := nameKey(players[i].PlayerName)
		if seen[key] {
			return invalidf("duplicate player name %q: player names must be unique in a campaign (case-insensitive)", players[i].PlayerName)
		}
		seen[key] = true
	}
	return nil
}

func prepareCampaignPlayer(player *domain.NewCampaignPlayer) error {
	if strings.TrimSpace(player.PlayerName) == "" {
		return invalidf("player name is required")
	}
	if strings.TrimSpace(player.Character.Name) == "" {
		return invalidf("character name is required for player %q", player.PlayerName)
	}
	player.Character.Name = strings.TrimSpace(player.Character.Name)
	if err := domain.ValidateCombatant(player.Character, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("player %q", player.PlayerName),
		RequireName: true,
		MinLevel:    1,
	}); err != nil {
		return domain.InvalidInput(err)
	}
	if strings.TrimSpace(player.Character.ID) == "" {
		player.Character.ID = uuid.NewString()
	}
	domain.NormalizeCombatantHP(&player.Character)
	player.Character.Side = domain.SideParty
	player.Character.XP = 0
	if player.Special.IsZero() {
		player.Special = domain.DefaultSpecialValues()
	}
	if err := player.Special.Validate(); err != nil {
		return invalidf("player %q: %w", player.PlayerName, err)
	}
	return nil
}
