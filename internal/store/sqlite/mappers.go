package sqlite

import (
	"strings"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

type combatantDBFields struct {
	ID                                string
	Name                              string
	Side                              domain.Side
	TorsoOnly                         int64
	Level                             int64
	XP                                int64
	Initiative                        int64
	HP                                int64
	MaxHP                             int64
	Defense                           int64
	DamageResistancePhysicalHead      int64
	DamageResistancePhysicalTorso     int64
	DamageResistancePhysicalLeftArm   int64
	DamageResistancePhysicalRightArm  int64
	DamageResistancePhysicalLeftLeg   int64
	DamageResistancePhysicalRightLeg  int64
	DamageResistanceEnergyHead        int64
	DamageResistanceEnergyTorso       int64
	DamageResistanceEnergyLeftArm     int64
	DamageResistanceEnergyRightArm    int64
	DamageResistanceEnergyLeftLeg     int64
	DamageResistanceEnergyRightLeg    int64
	DamageResistanceRadiationHead     int64
	DamageResistanceRadiationTorso    int64
	DamageResistanceRadiationLeftArm  int64
	DamageResistanceRadiationRightArm int64
	DamageResistanceRadiationLeftLeg  int64
	DamageResistanceRadiationRightLeg int64
	DamageResistancePhysical          int64
	DamageResistanceEnergy            int64
	DamageResistanceRadiation         int64
	DamageResistancePoison            int64
	DamageResistancePhysicalImmune    int64
	DamageResistanceEnergyImmune      int64
	DamageResistanceRadiationImmune   int64
	DamageResistancePoisonImmune      int64
	Active                            int64
	Defeated                          int64
}

func combatantFromFields(f combatantDBFields) domain.Combatant {
	return domain.Combatant{
		ID:                      f.ID,
		Name:                    f.Name,
		Side:                    f.Side,
		TorsoOnly:               f.TorsoOnly == 1,
		Level:                   int(f.Level),
		XP:                      int(f.XP),
		Initiative:              int(f.Initiative),
		HP:                      int(f.HP),
		MaxHP:                   int(f.MaxHP),
		Defense:                 int(f.Defense),
		ResistPhysicalHead:      int(f.DamageResistancePhysicalHead),
		ResistPhysicalTorso:     int(f.DamageResistancePhysicalTorso),
		ResistPhysicalLeftArm:   int(f.DamageResistancePhysicalLeftArm),
		ResistPhysicalRightArm:  int(f.DamageResistancePhysicalRightArm),
		ResistPhysicalLeftLeg:   int(f.DamageResistancePhysicalLeftLeg),
		ResistPhysicalRightLeg:  int(f.DamageResistancePhysicalRightLeg),
		ResistEnergyHead:        int(f.DamageResistanceEnergyHead),
		ResistEnergyTorso:       int(f.DamageResistanceEnergyTorso),
		ResistEnergyLeftArm:     int(f.DamageResistanceEnergyLeftArm),
		ResistEnergyRightArm:    int(f.DamageResistanceEnergyRightArm),
		ResistEnergyLeftLeg:     int(f.DamageResistanceEnergyLeftLeg),
		ResistEnergyRightLeg:    int(f.DamageResistanceEnergyRightLeg),
		ResistRadiationHead:     int(f.DamageResistanceRadiationHead),
		ResistRadiationTorso:    int(f.DamageResistanceRadiationTorso),
		ResistRadiationLeftArm:  int(f.DamageResistanceRadiationLeftArm),
		ResistRadiationRightArm: int(f.DamageResistanceRadiationRightArm),
		ResistRadiationLeftLeg:  int(f.DamageResistanceRadiationLeftLeg),
		ResistRadiationRightLeg: int(f.DamageResistanceRadiationRightLeg),
		ResistPhysical:          int(f.DamageResistancePhysical),
		ResistEnergy:            int(f.DamageResistanceEnergy),
		ResistRadiation:         int(f.DamageResistanceRadiation),
		ResistPoison:            int(f.DamageResistancePoison),
		ImmunePhysical:          f.DamageResistancePhysicalImmune == 1,
		ImmuneEnergy:            f.DamageResistanceEnergyImmune == 1,
		ImmuneRadiation:         f.DamageResistanceRadiationImmune == 1,
		ImmunePoison:            f.DamageResistancePoisonImmune == 1,
		Active:                  f.Active == 1,
		Defeated:                f.Defeated == 1,
	}
}

func combatantFromRow(r dbgen.ListCombatantsByEncounterIDRow) domain.Combatant {
	return combatantFromFields(combatantDBFields{
		ID:                                r.ID,
		Name:                              r.Name,
		Side:                              domain.Side(r.Side),
		TorsoOnly:                         r.TorsoOnly,
		Level:                             r.Level,
		XP:                                r.Xp,
		Initiative:                        r.Initiative,
		HP:                                r.Hp,
		MaxHP:                             r.MaxHp,
		Defense:                           r.Defense,
		DamageResistancePhysicalHead:      r.DamageResistancePhysicalHead,
		DamageResistancePhysicalTorso:     r.DamageResistancePhysicalTorso,
		DamageResistancePhysicalLeftArm:   r.DamageResistancePhysicalLeftArm,
		DamageResistancePhysicalRightArm:  r.DamageResistancePhysicalRightArm,
		DamageResistancePhysicalLeftLeg:   r.DamageResistancePhysicalLeftLeg,
		DamageResistancePhysicalRightLeg:  r.DamageResistancePhysicalRightLeg,
		DamageResistanceEnergyHead:        r.DamageResistanceEnergyHead,
		DamageResistanceEnergyTorso:       r.DamageResistanceEnergyTorso,
		DamageResistanceEnergyLeftArm:     r.DamageResistanceEnergyLeftArm,
		DamageResistanceEnergyRightArm:    r.DamageResistanceEnergyRightArm,
		DamageResistanceEnergyLeftLeg:     r.DamageResistanceEnergyLeftLeg,
		DamageResistanceEnergyRightLeg:    r.DamageResistanceEnergyRightLeg,
		DamageResistanceRadiationHead:     r.DamageResistanceRadiationHead,
		DamageResistanceRadiationTorso:    r.DamageResistanceRadiationTorso,
		DamageResistanceRadiationLeftArm:  r.DamageResistanceRadiationLeftArm,
		DamageResistanceRadiationRightArm: r.DamageResistanceRadiationRightArm,
		DamageResistanceRadiationLeftLeg:  r.DamageResistanceRadiationLeftLeg,
		DamageResistanceRadiationRightLeg: r.DamageResistanceRadiationRightLeg,
		DamageResistancePhysical:          r.DamageResistancePhysical,
		DamageResistanceEnergy:            r.DamageResistanceEnergy,
		DamageResistanceRadiation:         r.DamageResistanceRadiation,
		DamageResistancePoison:            r.DamageResistancePoison,
		DamageResistancePhysicalImmune:    r.DamageResistancePhysicalImmune,
		DamageResistanceEnergyImmune:      r.DamageResistanceEnergyImmune,
		DamageResistanceRadiationImmune:   r.DamageResistanceRadiationImmune,
		DamageResistancePoisonImmune:      r.DamageResistancePoisonImmune,
		Active:                            r.Active,
		Defeated:                          r.Defeated,
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
		ID:                                r.ID,
		Name:                              r.CharacterName,
		Side:                              domain.SideParty,
		TorsoOnly:                         r.TorsoOnly,
		Level:                             r.Level,
		Initiative:                        r.Initiative,
		HP:                                r.Hp,
		MaxHP:                             r.MaxHp,
		Defense:                           r.Defense,
		DamageResistancePhysicalHead:      r.DamageResistancePhysicalHead,
		DamageResistancePhysicalTorso:     r.DamageResistancePhysicalTorso,
		DamageResistancePhysicalLeftArm:   r.DamageResistancePhysicalLeftArm,
		DamageResistancePhysicalRightArm:  r.DamageResistancePhysicalRightArm,
		DamageResistancePhysicalLeftLeg:   r.DamageResistancePhysicalLeftLeg,
		DamageResistancePhysicalRightLeg:  r.DamageResistancePhysicalRightLeg,
		DamageResistanceEnergyHead:        r.DamageResistanceEnergyHead,
		DamageResistanceEnergyTorso:       r.DamageResistanceEnergyTorso,
		DamageResistanceEnergyLeftArm:     r.DamageResistanceEnergyLeftArm,
		DamageResistanceEnergyRightArm:    r.DamageResistanceEnergyRightArm,
		DamageResistanceEnergyLeftLeg:     r.DamageResistanceEnergyLeftLeg,
		DamageResistanceEnergyRightLeg:    r.DamageResistanceEnergyRightLeg,
		DamageResistanceRadiationHead:     r.DamageResistanceRadiationHead,
		DamageResistanceRadiationTorso:    r.DamageResistanceRadiationTorso,
		DamageResistanceRadiationLeftArm:  r.DamageResistanceRadiationLeftArm,
		DamageResistanceRadiationRightArm: r.DamageResistanceRadiationRightArm,
		DamageResistanceRadiationLeftLeg:  r.DamageResistanceRadiationLeftLeg,
		DamageResistanceRadiationRightLeg: r.DamageResistanceRadiationRightLeg,
		DamageResistancePhysical:          r.DamageResistancePhysical,
		DamageResistanceEnergy:            r.DamageResistanceEnergy,
		DamageResistanceRadiation:         r.DamageResistanceRadiation,
		DamageResistancePoison:            r.DamageResistancePoison,
		DamageResistancePhysicalImmune:    r.DamageResistancePhysicalImmune,
		DamageResistanceEnergyImmune:      r.DamageResistanceEnergyImmune,
		DamageResistanceRadiationImmune:   r.DamageResistanceRadiationImmune,
		DamageResistancePoisonImmune:      r.DamageResistancePoisonImmune,
	})
}

func partyCombatantsFromRows(rows []dbgen.ListActivePartyCharactersByCampaignIDRow) []domain.Combatant {
	party := make([]domain.Combatant, 0, len(rows))
	for _, r := range rows {
		party = append(party, partyCombatantFromRow(r))
	}
	return party
}

func campaignPlayerFromRow(r dbgen.ListActivePartyCharactersByCampaignIDRow) domain.NewCampaignPlayer {
	return domain.NewCampaignPlayer{
		PlayerName: r.PlayerName,
		Character:  partyCombatantFromRow(r),
	}
}

type encounterDBFields struct {
	ID         string
	CampaignID any
	Name       string
	Round      int64
	TurnIndex  int64
	PartyAP    int64
	GMThreat   int64
}

func encounterFromFields(f encounterDBFields, combatants []domain.Combatant) *domain.Encounter {
	return &domain.Encounter{
		ID:         f.ID,
		CampaignID: interfaceToString(f.CampaignID),
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
		CampaignID:      interfaceToString(r.CampaignID),
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
		ID:          c.ID,
		EncounterID: encounterID,
		Name:        c.Name,
		Side:        string(c.Side),
		TorsoOnly:   boolToInt64(c.TorsoOnly),
		Level:       int64(c.Level),
		Xp:          int64(c.XP),
		Initiative:  int64(c.Initiative),
		Hp:          int64(c.HP),
		MaxHp:       int64(c.MaxHP),
		Defense:     int64(c.Defense),
		Active:      boolToInt64(c.Active),
		Defeated:    boolToInt64(c.Defeated),
		Position:    int64(position),
	}
}

func insertPlayerCharacterParams(characterID, playerID, campaignID string, c domain.Combatant) dbgen.InsertPlayerCharacterParams {
	return dbgen.InsertPlayerCharacterParams{
		ID:         characterID,
		PlayerID:   playerID,
		CampaignID: campaignID,
		Name:       strings.TrimSpace(c.Name),
		Level:      int64(c.Level),
		Initiative: int64(c.Initiative),
		Hp:         int64(c.HP),
		MaxHp:      int64(c.MaxHP),
		Defense:    int64(c.Defense),
		TorsoOnly:  boolToInt64(c.TorsoOnly),
		Active:     1,
	}
}

func updateActivePlayerCharacterParams(characterID, campaignID string, c domain.Combatant) dbgen.UpdateActivePlayerCharacterByIDParams {
	return dbgen.UpdateActivePlayerCharacterByIDParams{
		CharacterID: characterID,
		CampaignID:  campaignID,
		Name:        strings.TrimSpace(c.Name),
		Level:       int64(c.Level),
		Initiative:  int64(c.Initiative),
		Hp:          int64(c.HP),
		MaxHp:       int64(c.MaxHP),
		Defense:     int64(c.Defense),
		TorsoOnly:   boolToInt64(c.TorsoOnly),
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
