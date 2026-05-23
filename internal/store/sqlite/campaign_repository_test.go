package sqlite

import (
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreUpdateCampaignKeepsInactiveCharacterHistory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.UpdateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "repo-char-2",
				Name:       "Ranger",
				Side:       domain.SideParty,
				Level:      2,
				Initiative: 8,
				HP:         8,
				MaxHP:      8,
				Defense:    2,
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(2), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))`,
		"repo-test-campaign",
		"Player 1",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?)) AND pc.active = 1`,
		"repo-test-campaign",
		"Player 1",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))
           AND lower(trim(pc.name)) = lower(trim(?)) AND pc.active = 0`,
		"repo-test-campaign",
		"Player 1",
		"Scout",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))
           AND lower(trim(pc.name)) = lower(trim(?)) AND pc.active = 1`,
		"repo-test-campaign",
		"Player 1",
		"Ranger",
	))

	party, err := store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, "Ranger", party[0].Name)
}

func TestEncounterStoreUpdateCampaignInactiveCharacterUnavailableAndRemovedFromEncounters(t *testing.T) {
	store := newTestStore(t)

	party, err := store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, party, 1)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "enc-with-party",
		Name: "Party Encounter",
		Combatants: []domain.Combatant{
			party[0],
			{ID: "npc-1", Name: "Raider", Side: domain.SideNPC, Level: 1, XP: 30, Initiative: 6, HP: 5, MaxHP: 5},
		},
	}))

	_, err = store.UpdateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Inactive:   true,
			Character: domain.Combatant{
				ID:         "repo-char-1",
				Name:       "Scout",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 7,
				HP:         6,
				MaxHP:      6,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)

	party, err = store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	assert.Empty(t, party)

	players, err := store.ListCampaignPlayers(t.Context(), "repo-test-campaign")
	require.NoError(t, err)
	require.Len(t, players, 1)
	assert.True(t, players[0].Inactive)

	enc, err := store.GetEncounterByID(t.Context(), "enc-with-party")
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 1)
	assert.Equal(t, "npc-1", enc.Combatants[0].ID)

	_, err = store.UpdateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "repo-char-1",
				Name:       "Scout",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 7,
				HP:         6,
				MaxHP:      6,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)

	party, err = store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, party, 1)

	enc, err = store.GetEncounterByID(t.Context(), "enc-with-party")
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 1)
	assert.Equal(t, "npc-1", enc.Combatants[0].ID)
}

func TestEncounterStoreListCampaignPlayersReadsNormalizedResistances(t *testing.T) {
	store := newTestStore(t)

	_, err := store.UpdateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:                      "repo-char-1",
				Name:                    "Scout",
				Side:                    domain.SideParty,
				Level:                   2,
				Initiative:              7,
				HP:                      6,
				MaxHP:                   6,
				Defense:                 1,
				ResistPhysical:          2,
				ResistEnergy:            3,
				ResistRadiation:         4,
				ResistPoison:            5,
				ResistPhysicalTorso:     6,
				ResistEnergyLeftArm:     7,
				ResistRadiationRightLeg: 8,
				ImmunePhysical:          true,
				ImmunePoison:            true,
			},
		},
	})
	require.NoError(t, err)

	players, err := store.ListCampaignPlayers(t.Context(), "repo-test-campaign")
	require.NoError(t, err)
	require.Len(t, players, 1)

	character := players[0].Character
	assert.Equal(t, 2, character.ResistPhysical)
	assert.Equal(t, 3, character.ResistEnergy)
	assert.Equal(t, 4, character.ResistRadiation)
	assert.Equal(t, 5, character.ResistPoison)
	assert.Equal(t, 6, character.ResistPhysicalTorso)
	assert.Equal(t, 7, character.ResistEnergyLeftArm)
	assert.Equal(t, 8, character.ResistRadiationRightLeg)
	assert.True(t, character.ImmunePhysical)
	assert.False(t, character.ImmuneEnergy)
	assert.False(t, character.ImmuneRadiation)
	assert.True(t, character.ImmunePoison)
}

func TestEncounterStoreMaintainsCampaignPlayerAuditFields(t *testing.T) {
	store := newTestStore(t)

	campaignFields := queryAuditFields(t, store.db, "campaigns", "repo-test-campaign")
	playerID := queryString(t, store.db, `SELECT id FROM players WHERE campaign_id = ? AND name = ?`, "repo-test-campaign", "Player 1")
	playerFields := queryAuditFields(t, store.db, "players", playerID)
	characterFields := queryAuditFields(t, store.db, "player_characters", "repo-char-1")

	assert.True(t, campaignFields.createdAt.Valid)
	assert.True(t, campaignFields.updatedAt.Valid)
	assert.False(t, campaignFields.deletedAt.Valid)
	assert.False(t, campaignFields.updatedAt.Time.Before(campaignFields.createdAt.Time))

	assert.True(t, playerFields.createdAt.Valid)
	assert.True(t, playerFields.updatedAt.Valid)
	assert.False(t, playerFields.deletedAt.Valid)
	assert.False(t, playerFields.updatedAt.Time.Before(playerFields.createdAt.Time))

	assert.True(t, characterFields.createdAt.Valid)
	assert.True(t, characterFields.updatedAt.Valid)
	assert.False(t, characterFields.deletedAt.Valid)
	assert.False(t, characterFields.updatedAt.Time.Before(characterFields.createdAt.Time))
}

func TestEncounterStoreStoresCampaignStartDateAsDateTime(t *testing.T) {
	store := newTestStore(t)
	expected := testCampaignStartDate(t)

	assert.Equal(t, "DATETIME", queryColumnType(t, store.db, "campaigns", "start_date"))

	var stored time.Time
	require.NoError(t, store.db.QueryRow(`SELECT start_date FROM campaigns WHERE id = ?`, "repo-test-campaign").Scan(&stored))
	assert.Equal(t, expected, stored)

	activeCampaign, err := store.GetActiveCampaign(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expected, activeCampaign.StartDate)
}

func TestEncounterStoreCreateCampaignRejectsZeroStartDate(t *testing.T) {
	store := newTestStore(t)

	_, err := store.CreateCampaign(t.Context(), "zero-date", "Zero Date", time.Time{}, []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				Name:       "Scout",
				Level:      1,
				Initiative: 7,
				HP:         6,
				MaxHP:      6,
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign start date is required")
}
