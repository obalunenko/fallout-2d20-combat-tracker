package app

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRevisionsAdvanceOnlyForAssistantOperations(t *testing.T) {
	svc := newSQLiteService(t)
	enc := newSQLiteEncounter(t, svc, 20)
	raider := npcID(t, enc)
	encKey, campKey := encounterKey(enc.ID), campaignKey("test-campaign")
	assistant := WithSource(t.Context(), domain.SourceAssistant)

	startEnc, startCamp := svc.Revision(encKey), svc.Revision(campKey)
	require.Equal(t, uint64(1), startEnc, "revisions start at 1")

	// Desktop operations never advance revisions.
	_, _, err := svc.ApplyDamage(t.Context(), raider, domain.DamagePhysical, domain.BodyTorso, 1)
	require.NoError(t, err)
	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	require.Equal(t, startEnc, svc.Revision(encKey))
	require.Equal(t, startCamp, svc.Revision(campKey))

	// An assistant encounter save advances the encounter, and the campaign
	// because a party combatant is present.
	_, _, err = svc.ApplyDamage(assistant, raider, domain.DamagePhysical, domain.BodyTorso, 1)
	require.NoError(t, err)
	require.Equal(t, startEnc+1, svc.Revision(encKey))
	require.Equal(t, startCamp+1, svc.Revision(campKey))

	// Activation and resource changes do not advance revisions.
	_, err = svc.ActivateEncounter(assistant, enc.ID)
	require.NoError(t, err)
	_, _, err = svc.AddPartyAP(assistant, 1)
	require.NoError(t, err)
	require.Equal(t, startEnc+1, svc.Revision(encKey))

	// An assistant campaign update advances the campaign and its encounters.
	players, err := svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	_, err = svc.UpdateCampaign(assistant, "test-campaign", "Renamed", testCampaignStartDate(t), players)
	require.NoError(t, err)
	require.Equal(t, startCamp+2, svc.Revision(campKey))
	require.Equal(t, startEnc+2, svc.Revision(encKey))
}

func TestStaleAndDeletedEditsAreRejected(t *testing.T) {
	svc := newSQLiteService(t)
	enc := newSQLiteEncounter(t, svc, 20)
	assistant := WithSource(t.Context(), domain.SourceAssistant)
	opened := svc.Revision(encounterKey(enc.ID))

	// The assistant changes the encounter after the editor opened.
	_, err := svc.UpdateEncounter(assistant, enc.ID, "Assistant Name", enc.Combatants)
	require.NoError(t, err)

	_, err = svc.ExecuteUpdateEncounter(t.Context(), UpdateEncounterCommand{
		EncounterID: enc.ID, Name: "Editor Name", Combatants: enc.Combatants, ExpectedRevision: opened,
	})
	require.ErrorIs(t, err, domain.ErrStaleEdit)

	// Overwrite: resubmit against the current revision.
	_, err = svc.ExecuteUpdateEncounter(t.Context(), UpdateEncounterCommand{
		EncounterID: enc.ID, Name: "Editor Name", Combatants: enc.Combatants, ExpectedRevision: svc.Revision(encounterKey(enc.ID)),
	})
	require.NoError(t, err)

	// 0 skips the check.
	_, err = svc.ExecuteUpdateEncounter(t.Context(), UpdateEncounterCommand{
		EncounterID: enc.ID, Name: "Unchecked", Combatants: enc.Combatants,
	})
	require.NoError(t, err)

	// The assistant deletes the encounter: the editor may only close.
	opened = svc.Revision(encounterKey(enc.ID))
	require.NoError(t, svc.DeleteEncounter(assistant, enc.ID))
	_, err = svc.ExecuteUpdateEncounter(t.Context(), UpdateEncounterCommand{
		EncounterID: enc.ID, Name: "Too Late", Combatants: enc.Combatants, ExpectedRevision: opened,
	})
	require.ErrorIs(t, err, domain.ErrEntityDeleted)
}

func TestStaleCampaignEditIsRejected(t *testing.T) {
	svc := newSQLiteService(t)
	players, err := svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	opened := svc.Revision(campaignKey("test-campaign"))

	_, err = svc.UpdateCampaign(WithSource(t.Context(), domain.SourceAssistant), "test-campaign", "By Assistant", testCampaignStartDate(t), players)
	require.NoError(t, err)

	_, err = svc.ExecuteUpdateCampaign(t.Context(), UpdateCampaignCommand{
		CampaignID: "test-campaign", Name: "By Editor", StartDate: testCampaignStartDate(t), Players: players, ExpectedRevision: opened,
	})
	require.ErrorIs(t, err, domain.ErrStaleEdit)
}
