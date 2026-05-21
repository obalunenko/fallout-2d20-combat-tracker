package sqlite

import (
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

const sqliteTimestampLayout = "2006-01-02 15:04:05.000"

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
	DefenseHead                       int64
	DefenseTorso                      int64
	DefenseLeftArm                    int64
	DefenseRightArm                   int64
	DefenseLeftLeg                    int64
	DefenseRightLeg                   int64
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
		DefenseHead:             int(f.DefenseHead),
		DefenseTorso:            int(f.DefenseTorso),
		DefenseLeftArm:          int(f.DefenseLeftArm),
		DefenseRightArm:         int(f.DefenseRightArm),
		DefenseLeftLeg:          int(f.DefenseLeftLeg),
		DefenseRightLeg:         int(f.DefenseRightLeg),
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
		DefenseHead:                       r.DefenseHead,
		DefenseTorso:                      r.DefenseTorso,
		DefenseLeftArm:                    r.DefenseLeftArm,
		DefenseRightArm:                   r.DefenseRightArm,
		DefenseLeftLeg:                    r.DefenseLeftLeg,
		DefenseRightLeg:                   r.DefenseRightLeg,
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
		DefenseHead:                       r.DefenseHead,
		DefenseTorso:                      r.DefenseTorso,
		DefenseLeftArm:                    r.DefenseLeftArm,
		DefenseRightArm:                   r.DefenseRightArm,
		DefenseLeftLeg:                    r.DefenseLeftLeg,
		DefenseRightLeg:                   r.DefenseRightLeg,
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

type campaignDBFields struct {
	ID        string
	Name      string
	StartDate string
	UpdatedAt time.Time
}

func campaignFromFields(f campaignDBFields) domain.Campaign {
	return domain.Campaign{
		ID:        f.ID,
		Name:      f.Name,
		StartDate: f.StartDate,
		UpdatedAt: f.UpdatedAt.Format(sqliteTimestampLayout),
	}
}

func campaignFromRow(r dbgen.GetActiveCampaignRow) domain.Campaign {
	return campaignFromFields(campaignDBFields{
		ID:        r.ID,
		Name:      r.Name,
		StartDate: r.StartDate,
		UpdatedAt: r.UpdatedAt,
	})
}

func campaignFromListRow(r dbgen.ListCampaignsRow) domain.Campaign {
	return campaignFromFields(campaignDBFields{
		ID:        r.ID,
		Name:      r.Name,
		StartDate: r.StartDate,
		UpdatedAt: r.UpdatedAt,
	})
}

func encounterLogFromRow(r dbgen.ListEncounterLogsByEncounterIDRow) domain.EncounterLog {
	return domain.EncounterLog{
		Round:     int(r.Round),
		Message:   r.Message,
		CreatedAt: r.CreatedAt.Format(sqliteTimestampLayout),
	}
}
