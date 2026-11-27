package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
)

// CampaignRosterPatch changes a campaign's details or one entry of its
// roster. Set exactly one of the patch fields.
type CampaignRosterPatch struct {
	CampaignID string

	Details         *CampaignDetailsPatch
	AddPlayer       *domain.NewCampaignPlayer
	UpdateCharacter *CharacterPatch
	SetAvailability *CharacterAvailabilityPatch
	RemovePlayer    *RemovePlayerPatch
}

// CampaignDetailsPatch changes a campaign's name or start date. Nil fields
// keep their stored value.
type CampaignDetailsPatch struct {
	Name      *string
	StartDate *time.Time
}

// CharacterPatch edits one player character.
type CharacterPatch struct {
	CharacterID string
	// Edit changes the stored player in place. It runs while the write gate
	// is held, so it must not call the Service. Changes to PlayerName and
	// Inactive are ignored: they have their own patches.
	Edit func(*domain.NewCampaignPlayer)
}

// CharacterAvailabilityPatch makes a character available or unavailable.
// An unavailable character is removed from every encounter of the campaign.
type CharacterAvailabilityPatch struct {
	CharacterID string
	Available   bool
}

// RemovePlayerPatch removes a player, and their character, from the roster.
// The character is also removed from the campaign's encounters.
type RemovePlayerPatch struct {
	CharacterID string
}

// CampaignRosterPatchResult is the stored state after a roster patch.
type CampaignRosterPatchResult struct {
	Campaign domain.Campaign
	// Player is the added or changed player. It is nil for Details and
	// RemovePlayer patches.
	Player *domain.NewCampaignPlayer
	// ReplacedCharacterID is the previous character ID when a rename created
	// a new character.
	ReplacedCharacterID string
}

// ExecuteCampaignRosterPatch reads the campaign roster, applies patch, and
// stores the full roster as ExecuteUpdateCampaign does, all under one write
// gate hold, so a desktop save cannot interleave between the read and the
// write. Players are matched by case-insensitive player name; renaming a
// character creates a new character ID.
func (s *Service) ExecuteCampaignRosterPatch(ctx context.Context, patch CampaignRosterPatch) (*CampaignRosterPatchResult, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()

	campaign, err := s.storedCampaign(ctx, patch.CampaignID)
	if err != nil {
		return nil, err
	}
	players, err := s.repo.ListCampaignPlayers(ctx, campaign.ID)
	if err != nil {
		return nil, err
	}
	cmd := UpdateCampaignCommand{CampaignID: campaign.ID, Name: campaign.Name, StartDate: campaign.StartDate, Players: players}

	var (
		playerKey  string // player to return, by name key
		replacedID string
	)
	switch {
	case patch.Details != nil:
		if patch.Details.Name != nil {
			cmd.Name = *patch.Details.Name
		}
		if patch.Details.StartDate != nil {
			cmd.StartDate = *patch.Details.StartDate
		}
	case patch.AddPlayer != nil:
		p := *patch.AddPlayer
		p.PlayerName = strings.TrimSpace(p.PlayerName)
		playerKey = nameKey(p.PlayerName)
		cmd.Players = append(cmd.Players, p)
	case patch.UpdateCharacter != nil:
		i, err := characterIndex(players, patch.UpdateCharacter.CharacterID)
		if err != nil {
			return nil, err
		}
		p := &cmd.Players[i]
		oldID, oldName, playerName, inactive := p.Character.ID, p.Character.Name, p.PlayerName, p.Inactive
		if patch.UpdateCharacter.Edit != nil {
			patch.UpdateCharacter.Edit(p)
		}
		p.PlayerName, p.Inactive, p.Character.ID = playerName, inactive, oldID
		if nameKey(p.Character.Name) != nameKey(oldName) {
			// The repository stores a renamed character as a new one.
			p.Character.ID = ""
			replacedID = oldID
		}
		playerKey = nameKey(playerName)
	case patch.SetAvailability != nil:
		i, err := characterIndex(players, patch.SetAvailability.CharacterID)
		if err != nil {
			return nil, err
		}
		cmd.Players[i].Inactive = !patch.SetAvailability.Available
		playerKey = nameKey(cmd.Players[i].PlayerName)
	case patch.RemovePlayer != nil:
		i, err := characterIndex(players, patch.RemovePlayer.CharacterID)
		if err != nil {
			return nil, err
		}
		if len(players) == 1 {
			return nil, invalidf("cannot remove the last player: a campaign needs at least one player")
		}
		// The store removes a dropped player's character from the
		// campaign's encounters in the same save.
		cmd.Players = slices.Delete(cmd.Players, i, i+1)
	default:
		return nil, invalidf("campaign patch is required")
	}

	if _, err := s.updateCampaign(ctx, op, cmd); err != nil {
		return nil, err
	}

	stored, err := s.storedCampaign(ctx, campaign.ID)
	if err != nil {
		return nil, err
	}
	res := &CampaignRosterPatchResult{Campaign: *stored, ReplacedCharacterID: replacedID}
	if playerKey == "" {
		return res, nil
	}
	players, err = s.repo.ListCampaignPlayers(ctx, campaign.ID)
	if err != nil {
		return nil, err
	}
	for i := range players {
		if nameKey(players[i].PlayerName) == playerKey {
			res.Player = &players[i]
			return res, nil
		}
	}
	return nil, fmt.Errorf("player %q not found after saving the campaign", playerKey)
}

// storedCampaign returns the stored campaign with the given ID. The caller
// holds the gate.
func (s *Service) storedCampaign(ctx context.Context, campaignID string) (*domain.Campaign, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, invalidf("campaign id is required")
	}
	campaigns, err := s.repo.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	for i := range campaigns {
		if campaigns[i].ID == campaignID {
			return &campaigns[i], nil
		}
	}
	return nil, domain.ErrCampaignNotFound
}

func characterIndex(players []domain.NewCampaignPlayer, characterID string) (int, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return -1, invalidf("character id is required")
	}
	i := slices.IndexFunc(players, func(p domain.NewCampaignPlayer) bool { return p.Character.ID == characterID })
	if i < 0 {
		return -1, notFoundf("character %q not found in this campaign", characterID)
	}
	return i, nil
}

// nameKey matches the repository's case-insensitive player matching.
func nameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
