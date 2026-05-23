package sqlite

import (
	"context"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

type bodyLocationID int64

const (
	bodyLocationHead bodyLocationID = iota + 1
	bodyLocationTorso
	bodyLocationLeftArm
	bodyLocationRightArm
	bodyLocationLeftLeg
	bodyLocationRightLeg
)

type damageTypeID int64

const (
	damageTypePhysical damageTypeID = iota + 1
	damageTypeEnergy
	damageTypeRadiation
	damageTypePoison
)

var damageTypeIDs = map[domain.DamageType]damageTypeID{
	domain.DamagePhysical:  damageTypePhysical,
	domain.DamageEnergy:    damageTypeEnergy,
	domain.DamageRadiation: damageTypeRadiation,
	domain.DamagePoison:    damageTypePoison,
}

var bodyLocationIDs = map[domain.BodyLocation]bodyLocationID{
	domain.BodyHead:     bodyLocationHead,
	domain.BodyTorso:    bodyLocationTorso,
	domain.BodyLeftArm:  bodyLocationLeftArm,
	domain.BodyRightArm: bodyLocationRightArm,
	domain.BodyLeftLeg:  bodyLocationLeftLeg,
	domain.BodyRightLeg: bodyLocationRightLeg,
}

type resistanceGlobalStat struct {
	damageTypeID damageTypeID
	resistance   int64
	immune       int64
}

type resistanceByLocationStat struct {
	damageTypeID   damageTypeID
	bodyLocationID bodyLocationID
	resistance     int64
}

func upsertCombatantNormalizedStats(ctx context.Context, qtx *dbgen.Queries, combatantID string, c domain.Combatant) error {
	return upsertNormalizedStats(
		c,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertCombatantResistanceGlobal(ctx, dbgen.UpsertCombatantResistanceGlobalParams{
				CombatantID:  combatantID,
				DamageTypeID: int64(stat.damageTypeID),
				Resistance:   stat.resistance,
				Immune:       stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertCombatantResistanceByLocation(ctx, dbgen.UpsertCombatantResistanceByLocationParams{
				CombatantID:    combatantID,
				DamageTypeID:   int64(stat.damageTypeID),
				BodyLocationID: int64(stat.bodyLocationID),
				Resistance:     stat.resistance,
			})
		},
	)
}

func upsertPlayerCharacterNormalizedStats(ctx context.Context, qtx *dbgen.Queries, playerCharacterID string, c domain.Combatant) error {
	return upsertNormalizedStats(
		c,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertPlayerCharacterResistanceGlobal(ctx, dbgen.UpsertPlayerCharacterResistanceGlobalParams{
				PlayerCharacterID: playerCharacterID,
				DamageTypeID:      int64(stat.damageTypeID),
				Resistance:        stat.resistance,
				Immune:            stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertPlayerCharacterResistanceByLocation(ctx, dbgen.UpsertPlayerCharacterResistanceByLocationParams{
				PlayerCharacterID: playerCharacterID,
				DamageTypeID:      int64(stat.damageTypeID),
				BodyLocationID:    int64(stat.bodyLocationID),
				Resistance:        stat.resistance,
			})
		},
	)
}

func upsertMonsterTemplateNormalizedStats(ctx context.Context, qtx *dbgen.Queries, monsterTemplateID string, c domain.Combatant) error {
	return upsertNormalizedStats(
		c,
		func(stat resistanceGlobalStat) error {
			return qtx.UpsertMonsterTemplateResistanceGlobal(ctx, dbgen.UpsertMonsterTemplateResistanceGlobalParams{
				MonsterTemplateID: monsterTemplateID,
				DamageTypeID:      int64(stat.damageTypeID),
				Resistance:        stat.resistance,
				Immune:            stat.immune,
			})
		},
		func(stat resistanceByLocationStat) error {
			return qtx.UpsertMonsterTemplateResistanceByLocation(ctx, dbgen.UpsertMonsterTemplateResistanceByLocationParams{
				MonsterTemplateID: monsterTemplateID,
				DamageTypeID:      int64(stat.damageTypeID),
				BodyLocationID:    int64(stat.bodyLocationID),
				Resistance:        stat.resistance,
			})
		},
	)
}

func upsertNormalizedStats(
	c domain.Combatant,
	upsertGlobalResistance func(resistanceGlobalStat) error,
	upsertLocationResistance func(resistanceByLocationStat) error,
) error {
	globalStats, err := globalResistanceStats(c)
	if err != nil {
		return fmt.Errorf("build global resistance stats: %w", err)
	}
	for _, stat := range globalStats {
		if err := upsertGlobalResistance(stat); err != nil {
			return fmt.Errorf("upsert global resistance: %w", err)
		}
	}

	locationStats, err := resistanceStatsByLocation(c)
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

func globalResistanceStats(c domain.Combatant) ([]resistanceGlobalStat, error) {
	profile := c.ResistanceProfile()
	stats := make([]resistanceGlobalStat, 0, len(domain.DamageTypes()))
	for _, damageType := range domain.DamageTypes() {
		damageTypeID, ok := damageTypeIDs[damageType]
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

func resistanceStatsByLocation(c domain.Combatant) ([]resistanceByLocationStat, error) {
	profile := c.ResistanceProfile()
	stats := make([]resistanceByLocationStat, 0, len(domain.LocationDamageTypes())*len(domain.BodyLocations()))
	for _, damageType := range domain.LocationDamageTypes() {
		damageTypeID, ok := damageTypeIDs[damageType]
		if !ok {
			return nil, fmt.Errorf("unknown damage type id: %q", damageType)
		}
		for _, bodyLocation := range domain.BodyLocations() {
			bodyLocationID, ok := bodyLocationIDs[bodyLocation]
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
