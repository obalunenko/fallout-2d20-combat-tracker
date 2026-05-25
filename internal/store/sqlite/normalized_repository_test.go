package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreSaveWritesNormalizedStatsWithoutTriggers(t *testing.T) {
	store := newTestStore(t)
	dropNormalizedSyncTriggers(t, store.db)

	combatant := domain.Combatant{
		ID:                      "norm-c1",
		Name:                    "Sentry",
		Side:                    domain.SideNPC,
		Initiative:              8,
		HP:                      10,
		MaxHP:                   10,
		ResistPhysical:          2,
		ResistEnergy:            3,
		ResistRadiation:         4,
		ResistPoison:            5,
		ResistPhysicalHead:      1,
		ResistPhysicalTorso:     2,
		ResistPhysicalLeftArm:   3,
		ResistPhysicalRightArm:  4,
		ResistPhysicalLeftLeg:   5,
		ResistPhysicalRightLeg:  6,
		ResistEnergyHead:        2,
		ResistEnergyTorso:       3,
		ResistEnergyLeftArm:     4,
		ResistEnergyRightArm:    5,
		ResistEnergyLeftLeg:     6,
		ResistEnergyRightLeg:    7,
		ResistRadiationHead:     3,
		ResistRadiationTorso:    4,
		ResistRadiationLeftArm:  5,
		ResistRadiationRightArm: 6,
		ResistRadiationLeftLeg:  7,
		ResistRadiationRightLeg: 8,
		ImmunePhysical:          true,
	}
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "norm-enc-1",
		Name:       "Normalized Save",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{combatant},
	}))

	assert.Equal(t, int64(2), queryInt64(t, store.db, `
			SELECT COUNT(*)
			FROM stat_profile_resistance_by_location
			WHERE stat_profile_id = ?
			  AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')
		`, statProfileID(statProfileCombatantKind, combatant.ID)))
	assert.Equal(t, int64(12), queryInt64(t, store.db, `
			SELECT COUNT(*)
			FROM stat_profile_resistance_by_location spr
			JOIN body_locations bl ON bl.id = spr.body_location_id
			WHERE spr.stat_profile_id = ?
		  AND bl.code <> 'global'
	`, statProfileID(statProfileCombatantKind, combatant.ID)))
	assert.Equal(
		t,
		int64(1),
		queryInt64(
			t,
			store.db,
			`SELECT immune
             FROM stat_profile_resistance_by_location
             WHERE stat_profile_id = ?
               AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
               AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')`,
			statProfileID(statProfileCombatantKind, combatant.ID),
			string(domain.DamagePhysical),
		),
	)
	assert.Equal(
		t,
		int64(5),
		queryInt64(
			t,
			store.db,
			`SELECT resistance
	             FROM stat_profile_resistance_by_location
	             WHERE stat_profile_id = ?
	               AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
	               AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')`,
			statProfileID(statProfileCombatantKind, combatant.ID),
			string(domain.DamagePoison),
		),
	)
}

func TestEncounterStoreCreateCampaignWritesNormalizedStatsWithoutTriggers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repo-norm-campaign.db")
	db, err := OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})
	dropNormalizedSyncTriggers(t, db)

	store := NewEncounterStore(db)
	characterID := "norm-char-1"
	_, err = store.CreateCampaign(t.Context(), "norm-campaign-1", "Normalized Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player A",
			Character: domain.Combatant{
				ID:                      characterID,
				Name:                    "Vera",
				Side:                    domain.SideParty,
				Level:                   2,
				Initiative:              9,
				HP:                      11,
				MaxHP:                   11,
				ResistEnergy:            3,
				ResistEnergyHead:        1,
				ResistEnergyTorso:       2,
				ResistEnergyLeftArm:     3,
				ResistEnergyRightArm:    4,
				ResistEnergyLeftLeg:     5,
				ResistEnergyRightLeg:    6,
				ImmuneRadiation:         true,
				ResistRadiation:         7,
				ResistRadiationHead:     1,
				ResistRadiationTorso:    1,
				ResistRadiationLeftArm:  1,
				ResistRadiationRightArm: 1,
				ResistRadiationLeftLeg:  1,
				ResistRadiationRightLeg: 1,
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(1), queryInt64(t, db, `
			SELECT COUNT(*)
			FROM stat_profile_resistance_by_location
			WHERE stat_profile_id = ?
			  AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')
		`, statProfileID(statProfilePlayerCharacterKind, characterID)))
	assert.Equal(t, int64(12), queryInt64(t, db, `
			SELECT COUNT(*)
			FROM stat_profile_resistance_by_location spr
			JOIN body_locations bl ON bl.id = spr.body_location_id
			WHERE spr.stat_profile_id = ?
		  AND bl.code <> 'global'
	`, statProfileID(statProfilePlayerCharacterKind, characterID)))
	assert.Equal(
		t,
		int64(0),
		queryInt64(
			t,
			db,
			`SELECT COUNT(*)
	             FROM stat_profile_resistance_by_location
	             WHERE stat_profile_id = ?
	               AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
	               AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')`,
			statProfileID(statProfilePlayerCharacterKind, characterID),
			string(domain.DamageEnergy),
		),
	)
	assert.Equal(
		t,
		int64(1),
		queryInt64(
			t,
			db,
			`SELECT immune
             FROM stat_profile_resistance_by_location
             WHERE stat_profile_id = ?
               AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
               AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')`,
			statProfileID(statProfilePlayerCharacterKind, characterID),
			string(domain.DamageRadiation),
		),
	)
}
