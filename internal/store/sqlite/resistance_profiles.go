package sqlite

import (
	"context"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

func combatantsByEncounterID(ctx context.Context, qtx *dbgen.Queries, encounterID string) ([]domain.Combatant, error) {
	rows, err := qtx.ListCombatantsByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, err
	}
	combatants := combatantsFromRows(rows)
	if len(combatants) == 0 {
		return combatants, nil
	}

	combatantProfiles, err := combatantResistanceProfiles(ctx, qtx, encounterID)
	if err != nil {
		return nil, err
	}
	linkedPlayerProfiles, err := linkedPlayerCharacterResistanceProfiles(ctx, qtx, encounterID)
	if err != nil {
		return nil, err
	}

	for i := range combatants {
		if profile, ok := linkedPlayerProfiles[combatants[i].ID]; ok {
			combatants[i].SetResistanceProfile(profile)
			continue
		}
		if profile, ok := combatantProfiles[combatants[i].ID]; ok {
			combatants[i].SetResistanceProfile(profile)
		}
	}
	return combatants, nil
}

func combatantResistanceProfiles(ctx context.Context, qtx *dbgen.Queries, encounterID string) (map[string]domain.ResistanceProfile, error) {
	globalRows, err := qtx.ListCombatantResistanceGlobalByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list combatant global resistances: %w", err)
	}
	locationRows, err := qtx.ListCombatantResistanceByLocationByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list combatant location resistances: %w", err)
	}

	return resistanceProfilesFromRows(
		globalRows,
		locationRows,
		func(row dbgen.ListCombatantResistanceGlobalByEncounterIDRow) (string, string, int64, int64) {
			return row.CombatantID, row.DamageType, row.Resistance, row.Immune
		},
		func(row dbgen.ListCombatantResistanceByLocationByEncounterIDRow) (string, string, string, int64) {
			return row.CombatantID, row.DamageType, row.BodyLocation, row.Resistance
		},
	), nil
}

func linkedPlayerCharacterResistanceProfiles(ctx context.Context, qtx *dbgen.Queries, encounterID string) (map[string]domain.ResistanceProfile, error) {
	globalRows, err := qtx.ListLinkedPlayerCharacterResistanceGlobalByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list linked player character global resistances: %w", err)
	}
	locationRows, err := qtx.ListLinkedPlayerCharacterResistanceByLocationByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list linked player character location resistances: %w", err)
	}

	return resistanceProfilesFromRows(
		globalRows,
		locationRows,
		func(row dbgen.ListLinkedPlayerCharacterResistanceGlobalByEncounterIDRow) (string, string, int64, int64) {
			return row.CombatantID, row.DamageType, row.Resistance, row.Immune
		},
		func(row dbgen.ListLinkedPlayerCharacterResistanceByLocationByEncounterIDRow) (string, string, string, int64) {
			return row.CombatantID, row.DamageType, row.BodyLocation, row.Resistance
		},
	), nil
}

func playerCharacterResistanceProfiles(ctx context.Context, qtx *dbgen.Queries, campaignID string) (map[string]domain.ResistanceProfile, error) {
	globalRows, err := qtx.ListActivePlayerCharacterResistanceGlobalByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list player character global resistances: %w", err)
	}
	locationRows, err := qtx.ListActivePlayerCharacterResistanceByLocationByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list player character location resistances: %w", err)
	}

	return resistanceProfilesFromRows(
		globalRows,
		locationRows,
		func(row dbgen.ListActivePlayerCharacterResistanceGlobalByCampaignIDRow) (string, string, int64, int64) {
			return row.PlayerCharacterID, row.DamageType, row.Resistance, row.Immune
		},
		func(row dbgen.ListActivePlayerCharacterResistanceByLocationByCampaignIDRow) (string, string, string, int64) {
			return row.PlayerCharacterID, row.DamageType, row.BodyLocation, row.Resistance
		},
	), nil
}

func monsterTemplateResistanceProfiles(ctx context.Context, qtx *dbgen.Queries) (map[string]domain.ResistanceProfile, error) {
	globalRows, err := qtx.ListMonsterTemplateResistanceGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monster template global resistances: %w", err)
	}
	locationRows, err := qtx.ListMonsterTemplateResistanceByLocation(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monster template location resistances: %w", err)
	}

	return resistanceProfilesFromRows(
		globalRows,
		locationRows,
		func(row dbgen.ListMonsterTemplateResistanceGlobalRow) (string, string, int64, int64) {
			return row.MonsterTemplateID, row.DamageType, row.Resistance, row.Immune
		},
		func(row dbgen.ListMonsterTemplateResistanceByLocationRow) (string, string, string, int64) {
			return row.MonsterTemplateID, row.DamageType, row.BodyLocation, row.Resistance
		},
	), nil
}

func resistanceProfilesFromRows[G any, L any](
	globalRows []G,
	locationRows []L,
	globalFields func(G) (string, string, int64, int64),
	locationFields func(L) (string, string, string, int64),
) map[string]domain.ResistanceProfile {
	profiles := make(map[string]domain.ResistanceProfile)
	for _, row := range globalRows {
		id, damageType, resistance, immune := globalFields(row)
		profile := resistanceProfile(profiles, id)
		profile.Global[domain.DamageType(damageType)] = domain.Resistance{
			Value:  int(resistance),
			Immune: immune == 1,
		}
	}
	for _, row := range locationRows {
		id, damageTypeCode, bodyLocation, resistance := locationFields(row)
		profile := resistanceProfile(profiles, id)
		damageType := domain.DamageType(damageTypeCode)
		byLocation := profile.ByLocation[damageType]
		if byLocation == nil {
			byLocation = make(map[domain.BodyLocation]int)
			profile.ByLocation[damageType] = byLocation
		}
		byLocation[domain.BodyLocation(bodyLocation)] = int(resistance)
	}
	return profiles
}

func resistanceProfile(profiles map[string]domain.ResistanceProfile, id string) domain.ResistanceProfile {
	profile, ok := profiles[id]
	if !ok {
		profile = domain.ResistanceProfile{
			Global:     make(map[domain.DamageType]domain.Resistance),
			ByLocation: make(map[domain.DamageType]map[domain.BodyLocation]int),
		}
		profiles[id] = profile
	}
	return profile
}

func applyPlayerCharacterResistanceProfile(combatant *domain.Combatant, profiles map[string]domain.ResistanceProfile) {
	if combatant == nil {
		return
	}
	profile, ok := profiles[combatant.PlayerCharacterID]
	if !ok {
		profile, ok = profiles[combatant.ID]
	}
	if !ok {
		return
	}
	combatant.SetResistanceProfile(profile)
}
