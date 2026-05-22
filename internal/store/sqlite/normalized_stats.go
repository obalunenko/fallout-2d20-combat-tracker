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
	for _, stat := range globalResistanceStats(c) {
		if err := upsertGlobalResistance(stat); err != nil {
			return fmt.Errorf("upsert global resistance: %w", err)
		}
	}

	for _, stat := range resistanceStatsByLocation(c) {
		if err := upsertLocationResistance(stat); err != nil {
			return fmt.Errorf("upsert location resistance: %w", err)
		}
	}

	return nil
}

func globalResistanceStats(c domain.Combatant) []resistanceGlobalStat {
	return []resistanceGlobalStat{
		{damageTypePhysical, int64(c.ResistPhysical), boolToInt64(c.ImmunePhysical)},
		{damageTypeEnergy, int64(c.ResistEnergy), boolToInt64(c.ImmuneEnergy)},
		{damageTypeRadiation, int64(c.ResistRadiation), boolToInt64(c.ImmuneRadiation)},
		{damageTypePoison, int64(c.ResistPoison), boolToInt64(c.ImmunePoison)},
	}
}

func resistanceStatsByLocation(c domain.Combatant) []resistanceByLocationStat {
	bodyLocationIDs := []bodyLocationID{
		bodyLocationHead,
		bodyLocationTorso,
		bodyLocationLeftArm,
		bodyLocationRightArm,
		bodyLocationLeftLeg,
		bodyLocationRightLeg,
	}
	resistanceByLocation := []struct {
		damageTypeID damageTypeID
		values       []int64
	}{
		{damageTypePhysical, []int64{
			int64(c.ResistPhysicalHead),
			int64(c.ResistPhysicalTorso),
			int64(c.ResistPhysicalLeftArm),
			int64(c.ResistPhysicalRightArm),
			int64(c.ResistPhysicalLeftLeg),
			int64(c.ResistPhysicalRightLeg),
		}},
		{damageTypeEnergy, []int64{
			int64(c.ResistEnergyHead),
			int64(c.ResistEnergyTorso),
			int64(c.ResistEnergyLeftArm),
			int64(c.ResistEnergyRightArm),
			int64(c.ResistEnergyLeftLeg),
			int64(c.ResistEnergyRightLeg),
		}},
		{damageTypeRadiation, []int64{
			int64(c.ResistRadiationHead),
			int64(c.ResistRadiationTorso),
			int64(c.ResistRadiationLeftArm),
			int64(c.ResistRadiationRightArm),
			int64(c.ResistRadiationLeftLeg),
			int64(c.ResistRadiationRightLeg),
		}},
	}

	stats := make([]resistanceByLocationStat, 0, len(resistanceByLocation)*len(bodyLocationIDs))
	for _, byDamageType := range resistanceByLocation {
		for idx, bodyLocationID := range bodyLocationIDs {
			stats = append(stats, resistanceByLocationStat{
				damageTypeID:   byDamageType.damageTypeID,
				bodyLocationID: bodyLocationID,
				resistance:     byDamageType.values[idx],
			})
		}
	}
	return stats
}
