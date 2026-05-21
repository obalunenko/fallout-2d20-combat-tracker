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
	Name                    string
	Side                    Side
	TorsoOnly               bool
	Level                   int
	XP                      int
	Initiative              int
	HP                      int
	MaxHP                   int
	Defense                 int
	DefenseHead             int
	DefenseTorso            int
	DefenseLeftArm          int
	DefenseRightArm         int
	DefenseLeftLeg          int
	DefenseRightLeg         int
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
	ID              string
	CampaignID      string
	Name            string
	Round           int
	Combatants      int
	UpdatedAt       time.Time
	Difficulty      string
	DifficultyScore float64
	PartyCount      int
	PartyAvgLevel   float64
	PartyXPBudget   int
	EnemyCount      int
	EnemyAvgLevel   float64
	EnemyTotalXP    int
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
	e.Resources.PartyAP += v
	if e.Resources.PartyAP < 0 {
		e.Resources.PartyAP = 0
	}
}

func (e *Encounter) SpendPartyAP(v int) error {
	if v < 0 {
		return fmt.Errorf("party AP spend value cannot be negative")
	}
	if e.Resources.PartyAP < v {
		return fmt.Errorf("not enough party AP")
	}
	e.Resources.PartyAP -= v
	return nil
}

func (e *Encounter) AddThreat(v int) {
	e.Resources.GMThreat += v
	if e.Resources.GMThreat < 0 {
		e.Resources.GMThreat = 0
	}
}

func (e *Encounter) SpendThreat(v int) error {
	if v < 0 {
		return fmt.Errorf("threat spend value cannot be negative")
	}
	if e.Resources.GMThreat < v {
		return fmt.Errorf("not enough threat")
	}
	e.Resources.GMThreat -= v
	return nil
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
	resistance, immune, err := target.resistanceByType(damageType)
	if err != nil {
		return 0, err
	}
	locationResistance, err := target.resistanceByTypeAndLocation(damageType, location)
	if err != nil {
		return 0, err
	}

	effective := 0
	if !immune {
		effective = max(amount-resistance-locationResistance, 0)
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

func (c *Combatant) resistanceByType(damageType DamageType) (int, bool, error) {
	switch damageType {
	case DamagePhysical:
		return 0, c.ImmunePhysical, nil
	case DamageEnergy:
		return 0, c.ImmuneEnergy, nil
	case DamageRadiation:
		return 0, c.ImmuneRadiation, nil
	case DamagePoison:
		return max(c.ResistPoison, 0), c.ImmunePoison, nil
	default:
		return 0, false, fmt.Errorf("unknown damage type: %q", damageType)
	}
}

func (c *Combatant) resistanceByTypeAndLocation(damageType DamageType, location BodyLocation) (int, error) {
	switch damageType {
	case DamagePhysical:
		return c.physicalLocationResistance(location)
	case DamageEnergy:
		return c.energyLocationResistance(location)
	case DamageRadiation:
		return c.radiationLocationResistance(location)
	case DamagePoison:
		// Poison is global-only, no body location reduction.
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown damage type: %q", damageType)
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
	c.normalizeLocationDefense()
	c.normalizeLocationResistance()
}

func NormalizeCombatantHP(c *Combatant) {
	if c == nil {
		return
	}
	c.normalizeHPState()
}

func (c *Combatant) normalizeLocationResistance() {
	c.ResistPhysical = max(c.ResistPhysical, 0)
	c.ResistEnergy = max(c.ResistEnergy, 0)
	c.ResistRadiation = max(c.ResistRadiation, 0)
	c.ResistPoison = max(c.ResistPoison, 0)
	c.ResistPhysicalHead = max(c.ResistPhysicalHead, 0)
	c.ResistPhysicalTorso = max(c.ResistPhysicalTorso, 0)
	c.ResistPhysicalLeftArm = max(c.ResistPhysicalLeftArm, 0)
	c.ResistPhysicalRightArm = max(c.ResistPhysicalRightArm, 0)
	c.ResistPhysicalLeftLeg = max(c.ResistPhysicalLeftLeg, 0)
	c.ResistPhysicalRightLeg = max(c.ResistPhysicalRightLeg, 0)
	c.ResistEnergyHead = max(c.ResistEnergyHead, 0)
	c.ResistEnergyTorso = max(c.ResistEnergyTorso, 0)
	c.ResistEnergyLeftArm = max(c.ResistEnergyLeftArm, 0)
	c.ResistEnergyRightArm = max(c.ResistEnergyRightArm, 0)
	c.ResistEnergyLeftLeg = max(c.ResistEnergyLeftLeg, 0)
	c.ResistEnergyRightLeg = max(c.ResistEnergyRightLeg, 0)
	c.ResistRadiationHead = max(c.ResistRadiationHead, 0)
	c.ResistRadiationTorso = max(c.ResistRadiationTorso, 0)
	c.ResistRadiationLeftArm = max(c.ResistRadiationLeftArm, 0)
	c.ResistRadiationRightArm = max(c.ResistRadiationRightArm, 0)
	c.ResistRadiationLeftLeg = max(c.ResistRadiationLeftLeg, 0)
	c.ResistRadiationRightLeg = max(c.ResistRadiationRightLeg, 0)
}

func (c *Combatant) normalizeLocationDefense() {
	c.DefenseHead = max(c.DefenseHead, 0)
	c.DefenseTorso = max(c.DefenseTorso, 0)
	c.DefenseLeftArm = max(c.DefenseLeftArm, 0)
	c.DefenseRightArm = max(c.DefenseRightArm, 0)
	c.DefenseLeftLeg = max(c.DefenseLeftLeg, 0)
	c.DefenseRightLeg = max(c.DefenseRightLeg, 0)
}

func (c *Combatant) physicalLocationResistance(location BodyLocation) (int, error) {
	switch location {
	case BodyHead:
		return max(c.ResistPhysicalHead, 0), nil
	case BodyTorso:
		return max(c.ResistPhysicalTorso, 0), nil
	case BodyLeftArm:
		return max(c.ResistPhysicalLeftArm, 0), nil
	case BodyRightArm:
		return max(c.ResistPhysicalRightArm, 0), nil
	case BodyLeftLeg:
		return max(c.ResistPhysicalLeftLeg, 0), nil
	case BodyRightLeg:
		return max(c.ResistPhysicalRightLeg, 0), nil
	default:
		return 0, fmt.Errorf("unknown body location: %q", location)
	}
}

func (c *Combatant) energyLocationResistance(location BodyLocation) (int, error) {
	switch location {
	case BodyHead:
		return max(c.ResistEnergyHead, 0), nil
	case BodyTorso:
		return max(c.ResistEnergyTorso, 0), nil
	case BodyLeftArm:
		return max(c.ResistEnergyLeftArm, 0), nil
	case BodyRightArm:
		return max(c.ResistEnergyRightArm, 0), nil
	case BodyLeftLeg:
		return max(c.ResistEnergyLeftLeg, 0), nil
	case BodyRightLeg:
		return max(c.ResistEnergyRightLeg, 0), nil
	default:
		return 0, fmt.Errorf("unknown body location: %q", location)
	}
}

func (c *Combatant) radiationLocationResistance(location BodyLocation) (int, error) {
	switch location {
	case BodyHead:
		return max(c.ResistRadiationHead, 0), nil
	case BodyTorso:
		return max(c.ResistRadiationTorso, 0), nil
	case BodyLeftArm:
		return max(c.ResistRadiationLeftArm, 0), nil
	case BodyRightArm:
		return max(c.ResistRadiationRightArm, 0), nil
	case BodyLeftLeg:
		return max(c.ResistRadiationLeftLeg, 0), nil
	case BodyRightLeg:
		return max(c.ResistRadiationRightLeg, 0), nil
	default:
		return 0, fmt.Errorf("unknown body location: %q", location)
	}
}

type EncounterDifficulty string

const (
	EncounterDifficultyUnknown EncounterDifficulty = "Unknown"
	EncounterDifficultyTrivial EncounterDifficulty = "Trivial"
	EncounterDifficultyEasy    EncounterDifficulty = "Easy"
	EncounterDifficultyNormal  EncounterDifficulty = "Normal"
	EncounterDifficultyHard    EncounterDifficulty = "Hard"
	EncounterDifficultyDeadly  EncounterDifficulty = "Deadly"
)

type EncounterDifficultyMetrics struct {
	Label         EncounterDifficulty
	Score         float64
	PartyCount    int
	PartyAvgLevel float64
	PartyXPBudget int
	EnemyCount    int
	EnemyAvgLevel float64
	EnemyTotalXP  int
}

func EvaluateEncounterDifficulty(combatants []Combatant) EncounterDifficultyMetrics {
	metrics := EncounterDifficultyMetrics{
		Label: EncounterDifficultyUnknown,
	}
	if len(combatants) == 0 {
		return metrics
	}

	partyLevelSum := 0
	enemyLevelSum := 0
	for i := range combatants {
		c := combatants[i]
		if c.Side == SideParty {
			metrics.PartyCount++
			partyLevelSum += c.Level
			continue
		}
		metrics.EnemyCount++
		enemyLevelSum += c.Level
		metrics.EnemyTotalXP += c.XP
	}

	if metrics.PartyCount > 0 {
		metrics.PartyAvgLevel = float64(partyLevelSum) / float64(metrics.PartyCount)
	}
	if metrics.EnemyCount > 0 {
		metrics.EnemyAvgLevel = float64(enemyLevelSum) / float64(metrics.EnemyCount)
	}

	if metrics.PartyCount == 0 || metrics.EnemyCount == 0 {
		return metrics
	}

	// Fallout 2d20 encounter budget baseline:
	// party budget = (average party level + 1) * number of players * 10
	// Difficulty compares total NPC XP against that budget.
	partyBudget := (metrics.PartyAvgLevel + 1) * float64(metrics.PartyCount) * 10
	if partyBudget <= 0 {
		return metrics
	}
	metrics.PartyXPBudget = int(math.Round(partyBudget))
	score := float64(metrics.EnemyTotalXP) / partyBudget
	metrics.Score = math.Round(score*100) / 100

	switch {
	case score < 0.5:
		metrics.Label = EncounterDifficultyTrivial
	case score < 1.0:
		metrics.Label = EncounterDifficultyEasy
	case score < 1.5:
		metrics.Label = EncounterDifficultyNormal
	case score <= 2.25:
		metrics.Label = EncounterDifficultyHard
	default:
		metrics.Label = EncounterDifficultyDeadly
	}
	return metrics
}
