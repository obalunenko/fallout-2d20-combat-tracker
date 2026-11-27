package app

import (
	"context"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestUserCausedErrorsAreMarked guards the adapter's error codes: every error
// a GM or assistant can cause through the Service must carry
// domain.ErrInvalidInput or domain.ErrNotFound, because the MCP adapter
// classifies by kind and reports anything unmarked as internal.
func TestUserCausedErrorsAreMarked(t *testing.T) {
	svc := newSQLiteService(t)
	enc := newSQLiteEncounter(t, svc, 20)
	raider := npcID(t, enc)
	start := testCampaignStartDate(t)
	pc := func(name string) domain.NewCampaignPlayer {
		return domain.NewCampaignPlayer{PlayerName: name, Character: domain.Combatant{Name: name, Side: domain.SideParty, Level: 1, HP: 5, MaxHP: 5}}
	}
	badSpecial := pc("Zed")
	badSpecial.Special = domain.SpecialValues{Strength: 0, Perception: 1, Endurance: 1, Charisma: 1, Intelligence: 1, Agility: 1, Luck: 1}
	overHP := pc("Hot")
	overHP.Character.HP = 9

	for _, tc := range []struct {
		name string
		kind error
		run  func(ctx context.Context) error
	}{
		{"campaign without name", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.CreateCampaign(ctx, "", "", start, []domain.NewCampaignPlayer{pc("A")})
			return err
		}},
		{"campaign without players", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.CreateCampaign(ctx, "", "Empty", start, nil)
			return err
		}},
		{"S.P.E.C.I.A.L. below 1", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.CreateCampaign(ctx, "", "Bad", start, []domain.NewCampaignPlayer{badSpecial})
			return err
		}},
		{"current HP above max HP", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.CreateCampaign(ctx, "", "Hot", start, []domain.NewCampaignPlayer{overHP})
			return err
		}},
		{"unknown campaign", domain.ErrNotFound, func(ctx context.Context) error {
			_, err := svc.ActivateCampaign(ctx, "no-such-campaign")
			return err
		}},
		{"encounter without combatants", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.CreateEncounter(ctx, "", "Empty", nil)
			return err
		}},
		{"party character twice in an encounter", domain.ErrInvalidInput, func(ctx context.Context) error {
			party := domain.Combatant{ID: "char-1", PlayerCharacterID: "char-1", Name: "Vault Dweller", Side: domain.SideParty, Level: 1, HP: 7, MaxHP: 7}
			_, err := svc.CreateEncounter(ctx, "", "Twice", []domain.Combatant{party, party})
			return err
		}},
		{"unknown encounter", domain.ErrNotFound, func(ctx context.Context) error {
			_, err := svc.ActivateEncounter(ctx, "no-such-encounter")
			return err
		}},
		{"damage to an unknown combatant", domain.ErrNotFound, func(ctx context.Context) error {
			_, _, err := svc.ApplyDamage(ctx, "ghost", domain.DamagePhysical, domain.BodyTorso, 1)
			return err
		}},
		{"negative damage", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, _, err := svc.ApplyDamage(ctx, raider, domain.DamagePhysical, domain.BodyTorso, -1)
			return err
		}},
		{"heal an unknown combatant", domain.ErrNotFound, func(ctx context.Context) error {
			_, _, err := svc.Heal(ctx, "ghost", 1)
			return err
		}},
		{"spend more threat than available", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.SpendThreat(ctx, 99)
			return err
		}},
		{"empty monster template", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.SaveMonsterTemplates(ctx, []domain.Combatant{{Name: ""}})
			return err
		}},
		{"activity limit out of range", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.ListAssistantActivity(ctx, 0)
			return err
		}},
		{"privileged port", domain.ErrInvalidInput, func(ctx context.Context) error {
			return svc.SaveAssistantSettings(ctx, domain.AssistantConnectionSettings{Port: 80, AccessSecret: "s"})
		}},
		{"unknown template in a draft", domain.ErrNotFound, func(ctx context.Context) error {
			_, err := svc.PreviewEncounterDifficulty(ctx, EncounterDraft{Monsters: []DraftMonster{{TemplateID: "nope", Quantity: 1}}})
			return err
		}},
		{"draft quantity out of range", domain.ErrInvalidInput, func(ctx context.Context) error {
			_, err := svc.PreviewEncounterDifficulty(ctx, EncounterDraft{Monsters: []DraftMonster{{Monster: &domain.Combatant{Name: "Rat", Level: 1, HP: 1, MaxHP: 1}, Quantity: 0}}})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t.Context())
			require.Error(t, err)
			require.ErrorIs(t, err, tc.kind, "message: %v", err)
		})
	}

	_, err := svc.SpendThreat(t.Context(), 99)
	require.EqualError(t, err, "not enough threat", "marking keeps the message the desktop shows")
}
