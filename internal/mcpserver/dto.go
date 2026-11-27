package mcpserver

import (
	"time"

	"github.com/obalunenko/fallout/internal/domain"
)

// DR is damage resistance per body location.
type DR struct {
	Head     int `json:"head" jsonschema:"DR on the head, 0 or more"`
	Torso    int `json:"torso" jsonschema:"DR on the torso, 0 or more"`
	LeftArm  int `json:"left_arm" jsonschema:"DR on the left arm, 0 or more"`
	RightArm int `json:"right_arm" jsonschema:"DR on the right arm, 0 or more"`
	LeftLeg  int `json:"left_leg" jsonschema:"DR on the left leg, 0 or more"`
	RightLeg int `json:"right_leg" jsonschema:"DR on the right leg, 0 or more"`
}

// Resistance is the Fallout 2d20 DR model: physical, energy and radiation DR
// per body location, poison DR global.
type Resistance struct {
	Physical  DR  `json:"physical" jsonschema:"physical DR per body location"`
	Energy    DR  `json:"energy" jsonschema:"energy DR per body location"`
	Radiation DR  `json:"radiation" jsonschema:"radiation DR per body location"`
	Poison    int `json:"poison" jsonschema:"global poison DR, 0 or more"`
}

// Immunity marks damage types that deal no damage at all.
type Immunity struct {
	Physical  bool `json:"physical"`
	Energy    bool `json:"energy"`
	Radiation bool `json:"radiation"`
	Poison    bool `json:"poison"`
}

// Special holds the seven S.P.E.C.I.A.L. attributes, each 1 or more.
type Special struct {
	Strength     int `json:"strength"`
	Perception   int `json:"perception"`
	Endurance    int `json:"endurance"`
	Charisma     int `json:"charisma"`
	Intelligence int `json:"intelligence"`
	Agility      int `json:"agility"`
	Luck         int `json:"luck"`
}

// Campaign is a campaign as the assistant sees it.
type Campaign struct {
	CampaignID string    `json:"campaign_id"`
	Name       string    `json:"name"`
	StartDate  string    `json:"start_date" jsonschema:"YYYY-MM-DD"`
	PartyAP    int       `json:"party_ap" jsonschema:"shared Party AP, 0 to 6"`
	GMThreat   int       `json:"gm_threat" jsonschema:"shared GM Threat, 0 or more"`
	IsActive   bool      `json:"is_active"`
	UpdatedAt  time.Time `json:"updated_at,omitzero"`
}

// Character is a campaign player character.
type Character struct {
	CharacterID string     `json:"character_id"`
	PlayerName  string     `json:"player_name"`
	Name        string     `json:"name"`
	Level       int        `json:"level"`
	Initiative  int        `json:"initiative"`
	HP          int        `json:"hp"`
	MaxHP       int        `json:"max_hp"`
	Defense     int        `json:"defense"`
	Resistance  Resistance `json:"resistance"`
	Immunity    Immunity   `json:"immunity"`
	Special     Special    `json:"special"`
	Notes       string     `json:"notes"`
	Available   bool       `json:"available" jsonschema:"false when the character is unavailable (stored as inactive)"`
}

// Combatant is one combatant in an encounter.
type Combatant struct {
	CombatantID   string     `json:"combatant_id"`
	CharacterID   string     `json:"character_id" jsonschema:"linked player character for party combatants, empty for NPCs"`
	Name          string     `json:"name"`
	Side          string     `json:"side" jsonschema:"party or npc"`
	TorsoOnly     bool       `json:"torso_only" jsonschema:"creature tracked on the torso only: every hit lands on the torso"`
	Level         int        `json:"level"`
	XP            int        `json:"xp"`
	Initiative    int        `json:"initiative"`
	HP            int        `json:"hp"`
	MaxHP         int        `json:"max_hp"`
	Defense       int        `json:"defense"`
	Resistance    Resistance `json:"resistance"`
	Immunity      Immunity   `json:"immunity"`
	Defeated      bool       `json:"defeated"`
	IsCurrentTurn bool       `json:"is_current_turn"`
}

// Difficulty is the encounter difficulty calculation.
type Difficulty struct {
	Label             string  `json:"label" jsonschema:"Trivial, Simple, Average, Hard, Deadly or Unknown"`
	UnavailableReason string  `json:"unavailable_reason,omitempty"`
	PartyCount        int     `json:"party_count"`
	AveragePCLevel    int     `json:"average_pc_level"`
	TotalMonsterXP    int     `json:"total_monster_xp"`
	XPBaseline        float64 `json:"xp_baseline"`
	EncounterLevel    int     `json:"encounter_level"`
	Difference        int     `json:"difference" jsonschema:"encounter level minus average PC level"`
}

// EncounterSummary is one row of the encounter list.
type EncounterSummary struct {
	EncounterID    string     `json:"encounter_id"`
	CampaignID     string     `json:"campaign_id"`
	Name           string     `json:"name"`
	Round          int        `json:"round"`
	CombatantCount int        `json:"combatant_count"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Difficulty     Difficulty `json:"difficulty"`
}

// Encounter is a full encounter.
type Encounter struct {
	EncounterID        string      `json:"encounter_id"`
	CampaignID         string      `json:"campaign_id"`
	Name               string      `json:"name"`
	Round              int         `json:"round"`
	TurnIndex          int         `json:"turn_index"`
	CurrentCombatantID string      `json:"current_combatant_id"`
	PartyAP            int         `json:"party_ap"`
	GMThreat           int         `json:"gm_threat"`
	Combatants         []Combatant `json:"combatants" jsonschema:"in initiative order"`
	Difficulty         Difficulty  `json:"difficulty"`
}

// MonsterTemplate is a saved monster.
type MonsterTemplate struct {
	TemplateID string     `json:"template_id"`
	Name       string     `json:"name"`
	TorsoOnly  bool       `json:"torso_only"`
	Level      int        `json:"level"`
	XP         int        `json:"xp"`
	Initiative int        `json:"initiative"`
	HP         int        `json:"hp"`
	MaxHP      int        `json:"max_hp"`
	Defense    int        `json:"defense"`
	Resistance Resistance `json:"resistance"`
	Immunity   Immunity   `json:"immunity"`
}

// LogEntry is one encounter operation-log entry.
type LogEntry struct {
	Round     int       `json:"round"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source" jsonschema:"desktop or assistant"`
}

// Activity is one assistant activity entry.
type Activity struct {
	ActivityID  string    `json:"activity_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	Operation   string    `json:"operation"`
	EntityKind  string    `json:"entity_kind,omitempty"`
	EntityID    string    `json:"entity_id,omitempty"`
	EntityLabel string    `json:"entity_label,omitempty"`
	Outcome     string    `json:"outcome" jsonschema:"done, rejected, declined, timed_out or refused"`
	Detail      string    `json:"detail,omitempty"`
}

// MonsterInput describes a monster to add or save as a template.
type MonsterInput struct {
	Name       string     `json:"name"`
	Level      int        `json:"level" jsonschema:"1 or more"`
	XP         int        `json:"xp" jsonschema:"XP awarded, 0 or more; the monsters' total XP drives encounter difficulty"`
	Initiative int        `json:"initiative" jsonschema:"higher acts first"`
	HP         int        `json:"hp" jsonschema:"current HP, 0 or more and no more than max_hp"`
	MaxHP      int        `json:"max_hp" jsonschema:"1 or more"`
	Defense    int        `json:"defense" jsonschema:"0 or more"`
	TorsoOnly  bool       `json:"torso_only,omitempty" jsonschema:"true for creatures tracked on the torso only, such as insects or limbless robots: every hit lands on the torso and uses torso DR, whatever location is named"`
	Resistance Resistance `json:"resistance" jsonschema:"DR per body location (head, torso, left and right arm, left and right leg) for physical, energy and radiation damage, plus one global poison DR; applied damage is raw damage minus the DR at the hit location. A torso-only creature needs only the torso values"`
	Immunity   Immunity   `json:"immunity" jsonschema:"damage types that deal this monster no damage"`
}

// DraftMonster adds copies of one monster to a draft roster. Set exactly one
// of TemplateID and Monster.
type DraftMonster struct {
	TemplateID string        `json:"template_id,omitempty" jsonschema:"a template_id from list_monster_templates; set this or monster, not both"`
	Monster    *MonsterInput `json:"monster,omitempty" jsonschema:"inline stats for a monster that has no template; set this or template_id, not both"`
	Quantity   int           `json:"quantity" jsonschema:"copies to add, 1 to 50; each copy is its own combatant and its XP counts toward difficulty"`
	Initiative *int          `json:"initiative,omitempty" jsonschema:"replaces the monster's initiative for these copies"`
}

// CharacterInput is a player character's name and combat stats.
type CharacterInput struct {
	Name       string      `json:"name"`
	Level      int         `json:"level" jsonschema:"1 or more"`
	Initiative int         `json:"initiative" jsonschema:"0 or more; higher acts first"`
	HP         int         `json:"hp" jsonschema:"current HP, 0 or more and no more than max_hp"`
	MaxHP      int         `json:"max_hp" jsonschema:"maximum HP, 1 or more"`
	Defense    int         `json:"defense" jsonschema:"0 or more"`
	Resistance *Resistance `json:"resistance,omitempty" jsonschema:"DR per body location for physical, energy and radiation damage, and global poison DR; omitted means 0 everywhere"`
	Immunity   *Immunity   `json:"immunity,omitempty" jsonschema:"damage types that deal no damage; omitted means none"`
}

// combatant returns the character as a party combatant.
func (in CharacterInput) combatant() domain.Combatant {
	c := domain.Combatant{
		Name:       in.Name,
		Side:       domain.SideParty,
		Level:      in.Level,
		Initiative: in.Initiative,
		HP:         in.HP,
		MaxHP:      in.MaxHP,
		Defense:    in.Defense,
	}
	var (
		r  Resistance
		im Immunity
	)
	if in.Resistance != nil {
		r = *in.Resistance
	}
	if in.Immunity != nil {
		im = *in.Immunity
	}
	applyResistance(&c, r, im)
	return c
}

func campaignDTO(c domain.Campaign, activeID string) Campaign {
	return Campaign{
		CampaignID: c.ID,
		Name:       c.Name,
		StartDate:  c.StartDate.Format(domain.CampaignDateLayout),
		PartyAP:    c.Resources.PartyAP,
		GMThreat:   c.Resources.GMThreat,
		IsActive:   c.ID == activeID,
		UpdatedAt:  c.UpdatedAt,
	}
}

func characterDTO(p domain.NewCampaignPlayer) Character {
	c := p.Character
	return Character{
		CharacterID: c.ID,
		PlayerName:  p.PlayerName,
		Name:        c.Name,
		Level:       c.Level,
		Initiative:  c.Initiative,
		HP:          c.HP,
		MaxHP:       c.MaxHP,
		Defense:     c.Defense,
		Resistance:  resistanceDTO(c),
		Immunity:    immunityDTO(c),
		Special:     specialDTO(p.Special),
		Notes:       p.Notes,
		Available:   !p.Inactive,
	}
}

func combatantDTO(c domain.Combatant, current bool) Combatant {
	characterID := ""
	if c.Side == domain.SideParty {
		characterID = c.PlayerCharacterID
	}
	return Combatant{
		CombatantID:   c.ID,
		CharacterID:   characterID,
		Name:          c.Name,
		Side:          string(c.Side),
		TorsoOnly:     c.TorsoOnly,
		Level:         c.Level,
		XP:            c.XP,
		Initiative:    c.Initiative,
		HP:            c.HP,
		MaxHP:         c.MaxHP,
		Defense:       c.Defense,
		Resistance:    resistanceDTO(c),
		Immunity:      immunityDTO(c),
		Defeated:      c.Defeated,
		IsCurrentTurn: current,
	}
}

func difficultyDTO(m domain.EncounterDifficultyMetrics) Difficulty {
	return Difficulty{
		Label:             string(m.Label),
		UnavailableReason: m.UnavailableReason,
		PartyCount:        m.PartyCount,
		AveragePCLevel:    m.AveragePCLevel,
		TotalMonsterXP:    m.TotalMonsterXP,
		XPBaseline:        m.XPBaseline,
		EncounterLevel:    m.EncounterLevel,
		Difference:        m.Difference,
	}
}

func encounterDTO(e *domain.Encounter) Encounter {
	out := Encounter{
		EncounterID: e.ID,
		CampaignID:  e.CampaignID,
		Name:        e.Name,
		Round:       e.Round,
		TurnIndex:   e.TurnIndex,
		PartyAP:     e.Resources.PartyAP,
		GMThreat:    e.Resources.GMThreat,
		Combatants:  make([]Combatant, 0, len(e.Combatants)),
		Difficulty:  difficultyDTO(domain.EvaluateEncounterDifficulty(e.Combatants)),
	}
	for i, c := range e.Combatants {
		current := i == e.TurnIndex
		if current {
			out.CurrentCombatantID = c.ID
		}
		out.Combatants = append(out.Combatants, combatantDTO(c, current))
	}
	return out
}

func encounterSummaryDTO(s domain.EncounterSummary) EncounterSummary {
	return EncounterSummary{
		EncounterID:    s.ID,
		CampaignID:     s.CampaignID,
		Name:           s.Name,
		Round:          s.Round,
		CombatantCount: s.Combatants,
		UpdatedAt:      s.UpdatedAt,
		Difficulty: Difficulty{
			Label:             s.Difficulty,
			UnavailableReason: s.DifficultyUnavailableReason,
			PartyCount:        s.PartyCount,
			AveragePCLevel:    s.AveragePCLevel,
			TotalMonsterXP:    s.TotalMonsterXP,
			XPBaseline:        s.XPBaseline,
			EncounterLevel:    s.EncounterLevel,
			Difference:        s.DifficultyDifference,
		},
	}
}

func monsterTemplateDTO(c domain.Combatant) MonsterTemplate {
	return MonsterTemplate{
		TemplateID: c.ID,
		Name:       c.Name,
		TorsoOnly:  c.TorsoOnly,
		Level:      c.Level,
		XP:         c.XP,
		Initiative: c.Initiative,
		HP:         c.HP,
		MaxHP:      c.MaxHP,
		Defense:    c.Defense,
		Resistance: resistanceDTO(c),
		Immunity:   immunityDTO(c),
	}
}

func logEntryDTO(l domain.EncounterLog) LogEntry {
	return LogEntry{Round: l.Round, Message: l.Message, CreatedAt: l.CreatedAt, Source: string(l.Source.Normalize())}
}

func activityDTO(a domain.AssistantActivityEntry) Activity {
	return Activity{
		ActivityID:  a.ID,
		OccurredAt:  a.OccurredAt,
		Operation:   a.Operation,
		EntityKind:  string(a.EntityKind),
		EntityID:    a.EntityID,
		EntityLabel: a.EntityLabel,
		Outcome:     string(a.Outcome),
		Detail:      a.Detail,
	}
}

func resistanceDTO(c domain.Combatant) Resistance {
	return Resistance{
		Physical:  DR{c.ResistPhysicalHead, c.ResistPhysicalTorso, c.ResistPhysicalLeftArm, c.ResistPhysicalRightArm, c.ResistPhysicalLeftLeg, c.ResistPhysicalRightLeg},
		Energy:    DR{c.ResistEnergyHead, c.ResistEnergyTorso, c.ResistEnergyLeftArm, c.ResistEnergyRightArm, c.ResistEnergyLeftLeg, c.ResistEnergyRightLeg},
		Radiation: DR{c.ResistRadiationHead, c.ResistRadiationTorso, c.ResistRadiationLeftArm, c.ResistRadiationRightArm, c.ResistRadiationLeftLeg, c.ResistRadiationRightLeg},
		Poison:    c.ResistPoison,
	}
}

func immunityDTO(c domain.Combatant) Immunity {
	return Immunity{Physical: c.ImmunePhysical, Energy: c.ImmuneEnergy, Radiation: c.ImmuneRadiation, Poison: c.ImmunePoison}
}

func specialDTO(v domain.SpecialValues) Special {
	return Special{v.Strength, v.Perception, v.Endurance, v.Charisma, v.Intelligence, v.Agility, v.Luck}
}

// applyResistance copies r and im onto c.
func applyResistance(c *domain.Combatant, r Resistance, im Immunity) {
	c.ResistPhysicalHead, c.ResistPhysicalTorso, c.ResistPhysicalLeftArm, c.ResistPhysicalRightArm, c.ResistPhysicalLeftLeg, c.ResistPhysicalRightLeg =
		r.Physical.Head, r.Physical.Torso, r.Physical.LeftArm, r.Physical.RightArm, r.Physical.LeftLeg, r.Physical.RightLeg
	c.ResistEnergyHead, c.ResistEnergyTorso, c.ResistEnergyLeftArm, c.ResistEnergyRightArm, c.ResistEnergyLeftLeg, c.ResistEnergyRightLeg =
		r.Energy.Head, r.Energy.Torso, r.Energy.LeftArm, r.Energy.RightArm, r.Energy.LeftLeg, r.Energy.RightLeg
	c.ResistRadiationHead, c.ResistRadiationTorso, c.ResistRadiationLeftArm, c.ResistRadiationRightArm, c.ResistRadiationLeftLeg, c.ResistRadiationRightLeg =
		r.Radiation.Head, r.Radiation.Torso, r.Radiation.LeftArm, r.Radiation.RightArm, r.Radiation.LeftLeg, r.Radiation.RightLeg
	c.ResistPoison = r.Poison
	c.ImmunePhysical, c.ImmuneEnergy, c.ImmuneRadiation, c.ImmunePoison = im.Physical, im.Energy, im.Radiation, im.Poison
}

func specialValues(s Special) domain.SpecialValues {
	return domain.SpecialValues{
		Strength: s.Strength, Perception: s.Perception, Endurance: s.Endurance, Charisma: s.Charisma,
		Intelligence: s.Intelligence, Agility: s.Agility, Luck: s.Luck,
	}
}

func monsterCombatant(m MonsterInput) domain.Combatant {
	c := domain.Combatant{
		Name:       m.Name,
		Side:       domain.SideNPC,
		TorsoOnly:  m.TorsoOnly,
		Level:      m.Level,
		XP:         m.XP,
		Initiative: m.Initiative,
		HP:         m.HP,
		MaxHP:      m.MaxHP,
		Defense:    m.Defense,
	}
	applyResistance(&c, m.Resistance, m.Immunity)
	return c
}
