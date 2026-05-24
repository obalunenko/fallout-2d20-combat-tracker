package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

const (
	playerCharacterAvailabilityActive   = "active"
	playerCharacterAvailabilityInactive = "inactive"
)

const (
	statProfileCombatantKind       = "combatant"
	statProfilePlayerCharacterKind = "player_character"
	statProfileMonsterTemplateKind = "monster_template"
)

func playerCharacterAvailabilityStatus(inactive bool) string {
	if inactive {
		return playerCharacterAvailabilityInactive
	}
	return playerCharacterAvailabilityActive
}

func statProfileID(kind, ownerID string) string {
	return kind + ":" + ownerID
}

type combatantDBFields struct {
	ID                string
	PlayerCharacterID string
	Name              string
	Side              domain.Side
	TorsoOnly         int64
	Level             int64
	XP                int64
	Initiative        int64
	HP                int64
	MaxHP             int64
	Defense           int64
	Active            int64
	Defeated          int64
}

func combatantFromFields(f combatantDBFields) domain.Combatant {
	return domain.Combatant{
		ID:                f.ID,
		PlayerCharacterID: f.PlayerCharacterID,
		Name:              f.Name,
		Side:              f.Side,
		TorsoOnly:         f.TorsoOnly == 1,
		Level:             int(f.Level),
		XP:                int(f.XP),
		Initiative:        int(f.Initiative),
		HP:                int(f.HP),
		MaxHP:             int(f.MaxHP),
		Defense:           int(f.Defense),
		Active:            f.Active == 1,
		Defeated:          f.Defeated == 1,
	}
}

func combatantFromRow(r dbgen.ListCombatantsByEncounterIDRow) domain.Combatant {
	return combatantFromFields(combatantDBFields{
		ID:                r.ID,
		PlayerCharacterID: interfaceToString(r.PlayerCharacterID),
		Name:              r.Name,
		Side:              domain.Side(r.Side),
		TorsoOnly:         r.TorsoOnly,
		Level:             r.Level,
		XP:                r.Xp,
		Initiative:        r.Initiative,
		HP:                r.Hp,
		MaxHP:             r.MaxHp,
		Defense:           r.Defense,
		Active:            r.Active,
		Defeated:          r.Defeated,
	})
}

func combatantsFromRows(rows []dbgen.ListCombatantsByEncounterIDRow) []domain.Combatant {
	combatants := make([]domain.Combatant, 0, len(rows))
	for _, r := range rows {
		combatants = append(combatants, combatantFromRow(r))
	}
	return combatants
}

func partyCombatantFromRow(r dbgen.ListActivePartyCharactersByCampaignIDRow) domain.Combatant {
	return combatantFromFields(combatantDBFields{
		ID:                r.ID,
		PlayerCharacterID: r.ID,
		Name:              r.CharacterName,
		Side:              domain.SideParty,
		TorsoOnly:         r.TorsoOnly,
		Level:             r.Level,
		Initiative:        r.Initiative,
		HP:                r.Hp,
		MaxHP:             r.MaxHp,
		Defense:           r.Defense,
	})
}

func campaignPlayerFromRow(r dbgen.ListActivePartyCharactersByCampaignIDRow) domain.NewCampaignPlayer {
	return domain.NewCampaignPlayer{
		PlayerName: r.PlayerName,
		Character:  partyCombatantFromRow(r),
		Inactive:   r.AvailabilityStatus == playerCharacterAvailabilityInactive,
	}
}

func monsterTemplateFromRow(r dbgen.ListMonsterTemplatesRow) domain.Combatant {
	return combatantFromFields(combatantDBFields{
		ID:         r.ID,
		Name:       r.Name,
		Side:       domain.SideNPC,
		TorsoOnly:  r.TorsoOnly,
		Level:      r.Level,
		XP:         r.Xp,
		Initiative: r.Initiative,
		HP:         r.Hp,
		MaxHP:      r.MaxHp,
		Defense:    r.Defense,
	})
}

type encounterDBFields struct {
	ID         string
	CampaignID string
	Name       string
	Round      int64
	TurnIndex  int64
	PartyAP    int64
	GMThreat   int64
}

func encounterFromFields(f encounterDBFields, combatants []domain.Combatant) *domain.Encounter {
	return &domain.Encounter{
		ID:         f.ID,
		CampaignID: f.CampaignID,
		Name:       f.Name,
		Round:      int(f.Round),
		TurnIndex:  int(f.TurnIndex),
		Combatants: combatants,
		Resources: domain.Resources{
			PartyAP:  int(f.PartyAP),
			GMThreat: int(f.GMThreat),
		},
	}
}

func encounterFromLatestRow(r dbgen.GetLatestEncounterByCampaignIDRow, combatants []domain.Combatant) *domain.Encounter {
	return encounterFromFields(encounterDBFields{
		ID:         r.ID,
		CampaignID: r.CampaignID,
		Name:       r.Name,
		Round:      r.Round,
		TurnIndex:  r.TurnIndex,
		PartyAP:    r.PartyAp,
		GMThreat:   r.GmThreat,
	}, combatants)
}

func encounterFromByIDRow(r dbgen.GetEncounterByIDByCampaignIDRow, combatants []domain.Combatant) *domain.Encounter {
	return encounterFromFields(encounterDBFields{
		ID:         r.ID,
		CampaignID: r.CampaignID,
		Name:       r.Name,
		Round:      r.Round,
		TurnIndex:  r.TurnIndex,
		PartyAP:    r.PartyAp,
		GMThreat:   r.GmThreat,
	}, combatants)
}

func encounterSummaryFromRow(r dbgen.ListEncounterSummariesByCampaignIDRow) domain.EncounterSummary {
	return domain.EncounterSummary{
		ID:              r.ID,
		CampaignID:      r.CampaignID,
		Name:            r.Name,
		Round:           int(r.Round),
		Combatants:      int(r.Combatants),
		Difficulty:      r.DifficultyLabel,
		DifficultyScore: r.DifficultyScore,
		PartyCount:      int(r.PartyCount),
		PartyAvgLevel:   r.PartyAvgLevel,
		PartyXPBudget:   int(r.PartyXpBudget),
		EnemyCount:      int(r.EnemyCount),
		EnemyAvgLevel:   r.EnemyAvgLevel,
		EnemyTotalXP:    int(r.EnemyTotalXp),
		UpdatedAt:       r.UpdatedAt,
	}
}

func insertCombatantParams(encounterID string, position int, c domain.Combatant) dbgen.InsertCombatantParams {
	return dbgen.InsertCombatantParams{
		ID:                c.ID,
		EncounterID:       encounterID,
		StatProfileID:     statProfileID(statProfileCombatantKind, c.ID),
		PlayerCharacterID: nullString(c.PlayerCharacterID),
		Name:              c.Name,
		Side:              string(c.Side),
		Active:            boolToInt64(c.Active),
		Defeated:          boolToInt64(c.Defeated),
		Position:          int64(position),
	}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func insertPlayerCharacterParams(characterID, playerID, campaignID string, c domain.Combatant, inactive bool) dbgen.InsertPlayerCharacterParams {
	return dbgen.InsertPlayerCharacterParams{
		ID:                 characterID,
		PlayerID:           playerID,
		CampaignID:         campaignID,
		StatProfileID:      statProfileID(statProfilePlayerCharacterKind, characterID),
		Name:               strings.TrimSpace(c.Name),
		Active:             1,
		AvailabilityStatus: playerCharacterAvailabilityStatus(inactive),
	}
}

func updateActivePlayerCharacterParams(characterID, campaignID string, c domain.Combatant, inactive bool) dbgen.UpdateActivePlayerCharacterByIDParams {
	return dbgen.UpdateActivePlayerCharacterByIDParams{
		CharacterID:        characterID,
		CampaignID:         campaignID,
		Name:               strings.TrimSpace(c.Name),
		AvailabilityStatus: playerCharacterAvailabilityStatus(inactive),
	}
}

func upsertMonsterTemplateParams(templateID string, c domain.Combatant) dbgen.UpsertMonsterTemplateParams {
	return dbgen.UpsertMonsterTemplateParams{
		ID:            templateID,
		StatProfileID: statProfileID(statProfileMonsterTemplateKind, templateID),
		Name:          strings.TrimSpace(c.Name),
		NameKey:       normalizeNameKey(c.Name),
	}
}

type campaignDBFields struct {
	ID        string
	Name      string
	StartDate time.Time
	UpdatedAt time.Time
}

func campaignFromFields(f campaignDBFields) domain.Campaign {
	return domain.Campaign{
		ID:        f.ID,
		Name:      f.Name,
		StartDate: f.StartDate,
		UpdatedAt: f.UpdatedAt,
	}
}

func campaignFromRow(r dbgen.GetActiveCampaignRow) domain.Campaign {
	return campaignFromFields(campaignDBFields{
		ID:        r.ID,
		Name:      r.Name,
		StartDate: truncateCampaignStartDate(r.StartDate),
		UpdatedAt: r.UpdatedAt,
	})
}

func campaignFromListRow(r dbgen.ListCampaignsRow) domain.Campaign {
	return campaignFromFields(campaignDBFields{
		ID:        r.ID,
		Name:      r.Name,
		StartDate: truncateCampaignStartDate(r.StartDate),
		UpdatedAt: r.UpdatedAt,
	})
}

func encounterLogFromRow(r dbgen.ListEncounterLogsByEncounterIDRow) domain.EncounterLog {
	return domain.EncounterLog{
		Round:     int(r.Round),
		Message:   r.Message,
		CreatedAt: r.CreatedAt,
	}
}

func truncateCampaignStartDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func formatCampaignStartDateForDB(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
