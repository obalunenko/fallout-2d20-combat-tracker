package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type Side string

const (
	SideParty Side = "party"
	SideNPC   Side = "npc"
)

type DamageType string

const (
	DamagePhysical  DamageType = "physical"
	DamageEnergy    DamageType = "energy"
	DamageRadiation DamageType = "radiation"
	DamagePoison    DamageType = "poison"
)

type BodyLocation string

const (
	BodyHead     BodyLocation = "head"
	BodyTorso    BodyLocation = "torso"
	BodyLeftArm  BodyLocation = "left_arm"
	BodyRightArm BodyLocation = "right_arm"
	BodyLeftLeg  BodyLocation = "left_leg"
	BodyRightLeg BodyLocation = "right_leg"
)

type Combatant struct {
	ID                      string
	PlayerCharacterID       string
	Name                    string
	Side                    Side
	TorsoOnly               bool
	Level                   int
	XP                      int
	Initiative              int
	HP                      int
	MaxHP                   int
	Defense                 int
	ResistPhysical          int
	ResistEnergy            int
	ResistRadiation         int
	ResistPoison            int
	ResistPhysicalHead      int
	ResistPhysicalTorso     int
	ResistPhysicalLeftArm   int
	ResistPhysicalRightArm  int
	ResistPhysicalLeftLeg   int
	ResistPhysicalRightLeg  int
	ResistEnergyHead        int
	ResistEnergyTorso       int
	ResistEnergyLeftArm     int
	ResistEnergyRightArm    int
	ResistEnergyLeftLeg     int
	ResistEnergyRightLeg    int
	ResistRadiationHead     int
	ResistRadiationTorso    int
	ResistRadiationLeftArm  int
	ResistRadiationRightArm int
	ResistRadiationLeftLeg  int
	ResistRadiationRightLeg int
	ImmunePhysical          bool
	ImmuneEnergy            bool
	ImmuneRadiation         bool
	ImmunePoison            bool
	Active                  bool
	Defeated                bool
}

type Resources struct {
	PartyAP  int
	GMThreat int
}

const MaxPartyAP = 6

func (r *Resources) Normalize() {
	if r.PartyAP < 0 {
		r.PartyAP = 0
	}
	if r.PartyAP > MaxPartyAP {
		r.PartyAP = MaxPartyAP
	}
	if r.GMThreat < 0 {
		r.GMThreat = 0
	}
}

func (r *Resources) AddPartyAP(v int) {
	r.PartyAP += v
	r.Normalize()
}

func (r *Resources) SpendPartyAP(v int) error {
	if v < 0 {
		return fmt.Errorf("party AP spend value cannot be negative")
	}
	if r.PartyAP < v {
		return fmt.Errorf("not enough party AP")
	}
	r.PartyAP -= v
	return nil
}

func (r *Resources) AddThreat(v int) {
	r.GMThreat += v
	r.Normalize()
}

func (r *Resources) SpendThreat(v int) error {
	if v < 0 {
		return fmt.Errorf("threat spend value cannot be negative")
	}
	if r.GMThreat < v {
		return fmt.Errorf("not enough threat")
	}
	r.GMThreat -= v
	return nil
}

type Encounter struct {
	ID         string
	CampaignID string
	Name       string
	Round      int
	TurnIndex  int
	Combatants []Combatant
	Resources  Resources
}

type EncounterSummary struct {
	ID                          string
	CampaignID                  string
	Name                        string
	Round                       int
	Combatants                  int
	UpdatedAt                   time.Time
	Difficulty                  string
	DifficultyUnavailableReason string
	PartyCount                  int
	AveragePCLevel              int
	TotalMonsterXP              int
	XPBaseline                  float64
	EncounterLevel              int
	DifficultyDifference        int
}

type EncounterLog struct {
	Round     int
	Message   string
	CreatedAt time.Time
}

func NewEncounter(id, name string, combatants []Combatant) *Encounter {
	cp := make([]Combatant, len(combatants))
	copy(cp, combatants)
	for i := range cp {
		cp[i].normalizeHPState()
	}

	sort.SliceStable(cp, func(i, j int) bool {
		if cp[i].Initiative == cp[j].Initiative {
			return cp[i].Name < cp[j].Name
		}
		return cp[i].Initiative > cp[j].Initiative
	})

	e := &Encounter{
		ID:         id,
		Name:       name,
		Round:      1,
		TurnIndex:  0,
		Combatants: cp,
		Resources: Resources{
			PartyAP:  0,
			GMThreat: 0,
		},
	}
	e.normalizeActive()
	return e
}

func (e *Encounter) ActiveCombatant() *Combatant {
	if len(e.Combatants) == 0 || e.TurnIndex < 0 || e.TurnIndex >= len(e.Combatants) {
		return nil
	}
	return &e.Combatants[e.TurnIndex]
}

func (e *Encounter) AdvanceTurn() error {
	if len(e.Combatants) == 0 {
		return fmt.Errorf("cannot advance turn: no combatants")
	}

	start := e.TurnIndex
	for {
		e.TurnIndex = (e.TurnIndex + 1) % len(e.Combatants)
		if e.TurnIndex == 0 {
			e.Round++
		}
		if !e.Combatants[e.TurnIndex].Defeated {
			break
		}
		if e.TurnIndex == start {
			return fmt.Errorf("cannot advance turn: all combatants are defeated")
		}
	}
	e.normalizeActive()
	return nil
}

func (e *Encounter) AddPartyAP(v int) {
	e.Resources.AddPartyAP(v)
}

func (e *Encounter) SpendPartyAP(v int) error {
	return e.Resources.SpendPartyAP(v)
}

func (e *Encounter) AddThreat(v int) {
	e.Resources.AddThreat(v)
}

func (e *Encounter) SpendThreat(v int) error {
	return e.Resources.SpendThreat(v)
}

func (e *Encounter) ApplyDamage(combatantID string, damageType DamageType, location BodyLocation, amount int) (int, error) {
	if combatantID == "" {
		return 0, fmt.Errorf("combatant id is required")
	}
	if amount < 0 {
		return 0, fmt.Errorf("damage amount cannot be negative")
	}

	combatantIdx := -1
	for i := range e.Combatants {
		if e.Combatants[i].ID == combatantID {
			combatantIdx = i
			break
		}
	}
	if combatantIdx < 0 {
		return 0, fmt.Errorf("combatant %q not found", combatantID)
	}

	target := &e.Combatants[combatantIdx]
	resistance, immune, err := target.damageResistance(damageType, location)
	if err != nil {
		return 0, err
	}

	effective := 0
	if !immune {
		effective = max(amount-resistance, 0)
	}

	target.HP -= effective
	if target.HP <= 0 {
		target.HP = 0
		target.Defeated = true
	}
	return effective, nil
}

func (e *Encounter) Heal(combatantID string, amount int) (int, error) {
	if combatantID == "" {
		return 0, fmt.Errorf("combatant id is required")
	}
	if amount < 0 {
		return 0, fmt.Errorf("heal amount cannot be negative")
	}

	combatantIdx := -1
	for i := range e.Combatants {
		if e.Combatants[i].ID == combatantID {
			combatantIdx = i
			break
		}
	}
	if combatantIdx < 0 {
		return 0, fmt.Errorf("combatant %q not found", combatantID)
	}

	target := &e.Combatants[combatantIdx]
	target.normalizeHPState()
	healed := amount
	if target.HP+healed > target.MaxHP {
		healed = target.MaxHP - target.HP
	}
	if healed < 0 {
		healed = 0
	}
	target.HP += healed
	if target.HP > 0 {
		target.Defeated = false
	}
	return healed, nil
}

func (e *Encounter) normalizeActive() {
	for i := range e.Combatants {
		e.Combatants[i].Active = i == e.TurnIndex
	}
}

func (c *Combatant) normalizeHPState() {
	if c.MaxHP <= 0 {
		if c.HP > 0 {
			c.MaxHP = c.HP
		} else if c.Defeated {
			c.MaxHP = 1
			c.HP = 0
		} else {
			c.MaxHP = 1
			c.HP = 1
		}
	}
	if c.HP < 0 {
		c.HP = 0
	}
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.HP == 0 {
		c.Defeated = true
	}
	c.normalizeLocationResistance()
}

func NormalizeCombatantHP(c *Combatant) {
	if c == nil {
		return
	}
	c.normalizeHPState()
}

func (c *Combatant) normalizeLocationResistance() {
	c.SetResistanceProfile(c.ResistanceProfile())
}

type EncounterDifficulty string

const (
	EncounterDifficultyUnknown EncounterDifficulty = "Unknown"
	EncounterDifficultyTrivial EncounterDifficulty = "Trivial"
	EncounterDifficultySimple  EncounterDifficulty = "Simple"
	EncounterDifficultyAverage EncounterDifficulty = "Average"
	EncounterDifficultyHard    EncounterDifficulty = "Hard"
	EncounterDifficultyDeadly  EncounterDifficulty = "Deadly"
)

type EncounterDifficultyMetrics struct {
	Label             EncounterDifficulty
	UnavailableReason string
	PartyCount        int
	AveragePCLevel    int
	TotalMonsterXP    int
	XPBaseline        float64
	EncounterLevel    int
	Difference        int
}

func EvaluateEncounterDifficulty(combatants []Combatant) EncounterDifficultyMetrics {
	metrics := EncounterDifficultyMetrics{
		Label:             EncounterDifficultyUnknown,
		UnavailableReason: "add at least one party member",
	}
	if len(combatants) == 0 {
		return metrics
	}

	partyLevelSum := 0
	for i := range combatants {
		c := combatants[i]
		if c.Side == SideParty {
			if c.Level < 1 {
				metrics.UnavailableReason = fmt.Sprintf("party member %q has invalid level", c.Name)
				return metrics
			}
			metrics.PartyCount++
			partyLevelSum += c.Level
			continue
		}
		if c.XP < 0 {
			metrics.UnavailableReason = fmt.Sprintf("monster %q has invalid XP", c.Name)
			return metrics
		}
		metrics.TotalMonsterXP += c.XP
	}
	if metrics.PartyCount == 0 {
		return metrics
	}

	metrics.AveragePCLevel = int(math.Ceil(float64(partyLevelSum) / float64(metrics.PartyCount)))
	metrics.XPBaseline = float64(metrics.TotalMonsterXP) / float64(metrics.PartyCount)
	metrics.EncounterLevel = int(math.Floor((metrics.XPBaseline - 10) / 10))
	if metrics.EncounterLevel < 1 {
		metrics.EncounterLevel = 1
	}
	metrics.Difference = metrics.EncounterLevel - metrics.AveragePCLevel
	metrics.UnavailableReason = ""

	switch {
	case metrics.Difference < -2:
		metrics.Label = EncounterDifficultyTrivial
	case metrics.Difference <= -1:
		metrics.Label = EncounterDifficultySimple
	case metrics.Difference <= 1:
		metrics.Label = EncounterDifficultyAverage
	case metrics.Difference <= 5:
		metrics.Label = EncounterDifficultyHard
	default:
		metrics.Label = EncounterDifficultyDeadly
	}
	return metrics
}
