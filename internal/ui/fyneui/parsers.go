package fyneui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/obalunenko/fallout/internal/domain"
)

func parseDamageType(v string) (domain.DamageType, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case string(domain.DamagePhysical):
		return domain.DamagePhysical, nil
	case string(domain.DamageEnergy):
		return domain.DamageEnergy, nil
	case string(domain.DamageRadiation):
		return domain.DamageRadiation, nil
	case string(domain.DamagePoison):
		return domain.DamagePoison, nil
	default:
		return "", fmt.Errorf("unknown damage type: %q", v)
	}
}

func parseBodyLocation(v string) (domain.BodyLocation, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case string(domain.BodyHead):
		return domain.BodyHead, nil
	case string(domain.BodyTorso):
		return domain.BodyTorso, nil
	case string(domain.BodyLeftArm):
		return domain.BodyLeftArm, nil
	case string(domain.BodyRightArm):
		return domain.BodyRightArm, nil
	case string(domain.BodyLeftLeg):
		return domain.BodyLeftLeg, nil
	case string(domain.BodyRightLeg):
		return domain.BodyRightLeg, nil
	default:
		return "", fmt.Errorf("unknown body location: %q", v)
	}
}

func parsePositiveIntOrError(raw, fieldName, label string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s for %q is required", fieldName, label)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid %s %q for %q", fieldName, raw, label)
	}
	return value, nil
}

func parseNonNegativeIntOrError(raw, fieldName, label string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s for %q is required", fieldName, label)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid %s %q for %q", fieldName, raw, label)
	}
	return value, nil
}

func parseResistanceCell(combatantName, resistType, raw string, immuneChecked bool) (int, bool, error) {
	if immuneChecked {
		return 0, true, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, fmt.Errorf("combatant %q: %s resistance is required", combatantName, resistType)
	}

	lower := strings.ToLower(value)
	if lower == "imm" || lower == "immune" || lower == "immunity" {
		return 0, true, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, false, fmt.Errorf("combatant %q: invalid %s resistance %q", combatantName, resistType, value)
	}
	return n, false, nil
}
