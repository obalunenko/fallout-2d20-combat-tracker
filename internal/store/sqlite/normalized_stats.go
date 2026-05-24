package sqlite

import (
	"context"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

type dictionaryIDs struct {
	damageTypes   map[domain.DamageType]int64
	bodyLocations map[domain.BodyLocation]int64
}

type resistanceGlobalStat struct {
	damageTypeID int64
	resistance   int64
	immune       int64
}

type resistanceByLocationStat struct {
	damageTypeID   int64
	bodyLocationID int64
	resistance     int64
}

func normalizedDictionaryIDs(ctx context.Context, qtx *dbgen.Queries) (dictionaryIDs, error) {
	damageTypeRows, err := qtx.ListDamageTypes(ctx)
	if err != nil {
		return dictionaryIDs{}, fmt.Errorf("list damage types: %w", err)
	}
	bodyLocationRows, err := qtx.ListBodyLocations(ctx)
	if err != nil {
		return dictionaryIDs{}, fmt.Errorf("list body locations: %w", err)
	}

	ids := dictionaryIDs{
		damageTypes:   make(map[domain.DamageType]int64, len(damageTypeRows)),
		bodyLocations: make(map[domain.BodyLocation]int64, len(bodyLocationRows)),
	}
	for _, row := range damageTypeRows {
		ids.damageTypes[domain.DamageType(row.Code)] = row.ID
	}
	for _, row := range bodyLocationRows {
		ids.bodyLocations[domain.BodyLocation(row.Code)] = row.ID
	}
	for _, damageType := range domain.DamageTypes() {
		if _, ok := ids.damageTypes[damageType]; !ok {
			return dictionaryIDs{}, fmt.Errorf("missing damage type dictionary row: %q", damageType)
		}
	}
	for _, bodyLocation := range domain.BodyLocations() {
		if _, ok := ids.bodyLocations[bodyLocation]; !ok {
			return dictionaryIDs{}, fmt.Errorf("missing body location dictionary row: %q", bodyLocation)
		}
	}
	return ids, nil
}

func upsertCombatantNormalizedStats(ctx context.Context, qtx *dbgen.Queries, ids dictionaryIDs, combatantID string, profile domain.CombatantProfile) error {
	return upsertNormalizedStats(
		ids,
		profile.Resistance,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertCombatantResistanceGlobal(ctx, dbgen.UpsertCombatantResistanceGlobalParams{
				CombatantID:  combatantID,
				DamageTypeID: stat.damageTypeID,
				Resistance:   stat.resistance,
				Immune:       stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertCombatantResistanceByLocation(ctx, dbgen.UpsertCombatantResistanceByLocationParams{
				CombatantID:    combatantID,
				DamageTypeID:   stat.damageTypeID,
				BodyLocationID: stat.bodyLocationID,
				Resistance:     stat.resistance,
			})
		},
	)
}

func upsertPlayerCharacterNormalizedStats(ctx context.Context, qtx *dbgen.Queries, ids dictionaryIDs, playerCharacterID string, profile domain.CombatantProfile) error {
	return upsertNormalizedStats(
		ids,
		profile.Resistance,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertPlayerCharacterResistanceGlobal(ctx, dbgen.UpsertPlayerCharacterResistanceGlobalParams{
				PlayerCharacterID: playerCharacterID,
				DamageTypeID:      stat.damageTypeID,
				Resistance:        stat.resistance,
				Immune:            stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertPlayerCharacterResistanceByLocation(ctx, dbgen.UpsertPlayerCharacterResistanceByLocationParams{
				PlayerCharacterID: playerCharacterID,
				DamageTypeID:      stat.damageTypeID,
				BodyLocationID:    stat.bodyLocationID,
				Resistance:        stat.resistance,
			})
		},
	)
}

func upsertMonsterTemplateNormalizedStats(ctx context.Context, qtx *dbgen.Queries, ids dictionaryIDs, monsterTemplateID string, profile domain.CombatantProfile) error {
	return upsertNormalizedStats(
		ids,
		profile.Resistance,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertMonsterTemplateResistanceGlobal(ctx, dbgen.UpsertMonsterTemplateResistanceGlobalParams{
				MonsterTemplateID: monsterTemplateID,
				DamageTypeID:      stat.damageTypeID,
				Resistance:        stat.resistance,
				Immune:            stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertMonsterTemplateResistanceByLocation(ctx, dbgen.UpsertMonsterTemplateResistanceByLocationParams{
				MonsterTemplateID: monsterTemplateID,
				DamageTypeID:      stat.damageTypeID,
				BodyLocationID:    stat.bodyLocationID,
				Resistance:        stat.resistance,
			})
		},
	)
}

func upsertNormalizedStats(
	ids dictionaryIDs,
	profile domain.ResistanceProfile,
	upsertGlobalResistance func(resistanceGlobalStat) error,
	upsertLocationResistance func(resistanceByLocationStat) error,
) error {
	globalStats, err := globalResistanceStats(ids, profile)
	if err != nil {
		return fmt.Errorf("build global resistance stats: %w", err)
	}
	for _, stat := range globalStats {
		if err := upsertGlobalResistance(stat); err != nil {
			return fmt.Errorf("upsert global resistance: %w", err)
		}
	}

	locationStats, err := resistanceStatsByLocation(ids, profile)
	if err != nil {
		return fmt.Errorf("build location resistance stats: %w", err)
	}
	for _, stat := range locationStats {
		if err := upsertLocationResistance(stat); err != nil {
			return fmt.Errorf("upsert location resistance: %w", err)
		}
	}

	return nil
}

func globalResistanceStats(ids dictionaryIDs, profile domain.ResistanceProfile) ([]resistanceGlobalStat, error) {
	stats := make([]resistanceGlobalStat, 0, len(domain.DamageTypes()))
	for _, damageType := range domain.DamageTypes() {
		damageTypeID, ok := ids.damageTypes[damageType]
		if !ok {
			return nil, fmt.Errorf("unknown damage type id: %q", damageType)
		}
		resistance, immune, err := profile.GlobalResistance(damageType)
		if err != nil {
			return nil, err
		}
		stats = append(stats, resistanceGlobalStat{
			damageTypeID: damageTypeID,
			resistance:   int64(resistance),
			immune:       boolToInt64(immune),
		})
	}
	return stats, nil
}

func resistanceStatsByLocation(ids dictionaryIDs, profile domain.ResistanceProfile) ([]resistanceByLocationStat, error) {
	stats := make([]resistanceByLocationStat, 0, len(domain.LocationDamageTypes())*len(domain.BodyLocations()))
	for _, damageType := range domain.LocationDamageTypes() {
		damageTypeID, ok := ids.damageTypes[damageType]
		if !ok {
			return nil, fmt.Errorf("unknown damage type id: %q", damageType)
		}
		for _, bodyLocation := range domain.BodyLocations() {
			bodyLocationID, ok := ids.bodyLocations[bodyLocation]
			if !ok {
				return nil, fmt.Errorf("unknown body location id: %q", bodyLocation)
			}
			resistance, err := profile.LocationResistance(damageType, bodyLocation)
			if err != nil {
				return nil, err
			}
			stats = append(stats, resistanceByLocationStat{
				damageTypeID:   damageTypeID,
				bodyLocationID: bodyLocationID,
				resistance:     int64(resistance),
			})
		}
	}
	return stats, nil
}
