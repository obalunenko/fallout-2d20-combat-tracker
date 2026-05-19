package domain

import (
	"fmt"
	"sort"
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

type Combatant struct {
	ID              string
	Name            string
	Side            Side
	Level           int
	XP              int
	Initiative      int
	HP              int
	Defense         int
	ResistPhysical  int
	ResistEnergy    int
	ResistRadiation int
	ResistPoison    int
	ImmunePhysical  bool
	ImmuneEnergy    bool
	ImmuneRadiation bool
	ImmunePoison    bool
	Active          bool
	Defeated        bool
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
	ID         string
	CampaignID string
	Name       string
	Round      int
	Combatants int
	UpdatedAt  string
}

type EncounterLog struct {
	Round     int
	Message   string
	CreatedAt string
}

func NewEncounter(id, name string, combatants []Combatant) *Encounter {
	cp := make([]Combatant, len(combatants))
	copy(cp, combatants)

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

func (e *Encounter) ApplyDamage(combatantID string, damageType DamageType, amount int) (int, error) {
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
	target.HP += amount
	if target.HP > 0 {
		target.Defeated = false
	}
	return amount, nil
}

func (e *Encounter) normalizeActive() {
	for i := range e.Combatants {
		e.Combatants[i].Active = i == e.TurnIndex
	}
}

func (c *Combatant) resistanceByType(damageType DamageType) (int, bool, error) {
	switch damageType {
	case DamagePhysical:
		return c.ResistPhysical, c.ImmunePhysical, nil
	case DamageEnergy:
		return c.ResistEnergy, c.ImmuneEnergy, nil
	case DamageRadiation:
		return c.ResistRadiation, c.ImmuneRadiation, nil
	case DamagePoison:
		return c.ResistPoison, c.ImmunePoison, nil
	default:
		return 0, false, fmt.Errorf("unknown damage type: %q", damageType)
	}
}
