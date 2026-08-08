package domain

import (
	"fmt"
	"time"
)

type SpecialAttribute string

const (
	SpecialStrength     SpecialAttribute = "strength"
	SpecialPerception   SpecialAttribute = "perception"
	SpecialEndurance    SpecialAttribute = "endurance"
	SpecialCharisma     SpecialAttribute = "charisma"
	SpecialIntelligence SpecialAttribute = "intelligence"
	SpecialAgility      SpecialAttribute = "agility"
	SpecialLuck         SpecialAttribute = "luck"
)

func SpecialAttributes() []SpecialAttribute {
	return []SpecialAttribute{
		SpecialStrength,
		SpecialPerception,
		SpecialEndurance,
		SpecialCharisma,
		SpecialIntelligence,
		SpecialAgility,
		SpecialLuck,
	}
}

type SpecialValues struct {
	Strength     int
	Perception   int
	Endurance    int
	Charisma     int
	Intelligence int
	Agility      int
	Luck         int
}

func DefaultSpecialValues() SpecialValues {
	return SpecialValues{
		Strength:     1,
		Perception:   1,
		Endurance:    1,
		Charisma:     1,
		Intelligence: 1,
		Agility:      1,
		Luck:         1,
	}
}

func (v SpecialValues) IsZero() bool {
	return v == (SpecialValues{})
}

func (v SpecialValues) Value(attribute SpecialAttribute) int {
	switch attribute {
	case SpecialStrength:
		return v.Strength
	case SpecialPerception:
		return v.Perception
	case SpecialEndurance:
		return v.Endurance
	case SpecialCharisma:
		return v.Charisma
	case SpecialIntelligence:
		return v.Intelligence
	case SpecialAgility:
		return v.Agility
	case SpecialLuck:
		return v.Luck
	default:
		return 0
	}
}

func (v *SpecialValues) Set(attribute SpecialAttribute, value int) error {
	if v == nil {
		return fmt.Errorf("S.P.E.C.I.A.L. values cannot be nil")
	}
	switch attribute {
	case SpecialStrength:
		v.Strength = value
	case SpecialPerception:
		v.Perception = value
	case SpecialEndurance:
		v.Endurance = value
	case SpecialCharisma:
		v.Charisma = value
	case SpecialIntelligence:
		v.Intelligence = value
	case SpecialAgility:
		v.Agility = value
	case SpecialLuck:
		v.Luck = value
	default:
		return fmt.Errorf("unknown S.P.E.C.I.A.L. attribute %q", attribute)
	}
	return nil
}

func (v SpecialValues) Validate() error {
	for _, attribute := range SpecialAttributes() {
		if v.Value(attribute) < 1 {
			return fmt.Errorf("%s must be at least 1", attribute)
		}
	}
	return nil
}

type Campaign struct {
	ID        string
	Name      string
	StartDate time.Time
	Resources Resources
	UpdatedAt time.Time
}

type NewCampaignPlayer struct {
	PlayerName string
	Character  Combatant
	Notes      string
	Special    SpecialValues
	Inactive   bool
}

type CampaignCharacter struct {
	PlayerID   string
	PlayerName string
	Character  Combatant
	Active     bool
}
