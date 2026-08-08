package sqlite

import (
	"context"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

func specialAttributeIDs(ctx context.Context, qtx *dbgen.Queries) (map[domain.SpecialAttribute]int64, error) {
	rows, err := qtx.ListSpecialAttributes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list SPECIAL attributes: %w", err)
	}
	ids := make(map[domain.SpecialAttribute]int64, len(rows))
	for _, row := range rows {
		ids[domain.SpecialAttribute(row.Code)] = row.ID
	}
	for _, attribute := range domain.SpecialAttributes() {
		if _, ok := ids[attribute]; !ok {
			return nil, fmt.Errorf("SPECIAL attribute %q is not configured", attribute)
		}
	}
	return ids, nil
}

func upsertPlayerCharacterSpecialValues(ctx context.Context, qtx *dbgen.Queries, ids map[domain.SpecialAttribute]int64, characterID string, values domain.SpecialValues) error {
	if err := values.Validate(); err != nil {
		return err
	}
	for _, attribute := range domain.SpecialAttributes() {
		if err := qtx.UpsertPlayerCharacterSpecialValue(ctx, dbgen.UpsertPlayerCharacterSpecialValueParams{
			PlayerCharacterID:  characterID,
			SpecialAttributeID: ids[attribute],
			Value:              int64(values.Value(attribute)),
		}); err != nil {
			return fmt.Errorf("save %s: %w", attribute, err)
		}
	}
	return nil
}

func playerCharacterSpecialValuesByCampaign(ctx context.Context, qtx *dbgen.Queries, campaignID string) (map[string]domain.SpecialValues, error) {
	rows, err := qtx.ListActivePlayerCharacterSpecialValuesByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list player character SPECIAL values: %w", err)
	}
	valuesByCharacter := make(map[string]domain.SpecialValues)
	counts := make(map[string]int)
	for _, row := range rows {
		values := valuesByCharacter[row.PlayerCharacterID]
		if err := values.Set(domain.SpecialAttribute(row.SpecialAttribute), int(row.Value)); err != nil {
			return nil, err
		}
		valuesByCharacter[row.PlayerCharacterID] = values
		counts[row.PlayerCharacterID]++
	}
	for characterID, values := range valuesByCharacter {
		if counts[characterID] != len(domain.SpecialAttributes()) {
			return nil, fmt.Errorf("player character %s has incomplete SPECIAL values", characterID)
		}
		if err := values.Validate(); err != nil {
			return nil, fmt.Errorf("player character %s: %w", characterID, err)
		}
	}
	return valuesByCharacter, nil
}
