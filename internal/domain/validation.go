package domain

import (
	"fmt"
	"strings"
)

type CombatantValidationOptions struct {
	Label       string
	RequireName bool
	RequireSide bool
	MinLevel    int
}

func ValidateCombatant(c Combatant, opts CombatantValidationOptions) error {
	prefix := validationPrefix(opts.Label)
	if opts.RequireName && strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%sname is required", prefix)
	}
	if opts.RequireSide || c.Side != "" {
		if c.Side != SideParty && c.Side != SideNPC {
			return fmt.Errorf("%sinvalid side", prefix)
		}
	}
	if c.Level < opts.MinLevel {
		return fmt.Errorf("%sinvalid level", prefix)
	}
	if c.XP < 0 {
		return fmt.Errorf("%sinvalid XP", prefix)
	}
	if c.Initiative < 0 {
		return fmt.Errorf("%sinvalid initiative", prefix)
	}
	if c.HP < 0 {
		return fmt.Errorf("%sinvalid HP", prefix)
	}
	if c.MaxHP < 0 {
		return fmt.Errorf("%sinvalid max HP", prefix)
	}
	if c.MaxHP > 0 && c.HP > c.MaxHP {
		return fmt.Errorf("%scurrent HP cannot exceed max HP", prefix)
	}
	if c.Defense < 0 {
		return fmt.Errorf("%sinvalid defense", prefix)
	}
	if err := ValidateResistanceProfile(c.resistanceProfile(false)); err != nil {
		return fmt.Errorf("%s%w", prefix, err)
	}
	return nil
}

func ValidateResistanceProfile(profile ResistanceProfile) error {
	if profile.HasNegativeValues() {
		return fmt.Errorf("invalid resistance")
	}
	return nil
}

func validationPrefix(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return label + ": "
}
