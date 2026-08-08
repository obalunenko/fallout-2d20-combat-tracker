package fyneui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/obalunenko/fallout/internal/domain"
)

func collectCombatantsFromRows(rows []*combatantInputRow) ([]domain.Combatant, error) {
	combatants := make([]domain.Combatant, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.name.Text)
		if name == "" {
			continue
		}

		initiativeText := strings.TrimSpace(row.initiative.Text)
		if initiativeText == "" {
			return nil, fmt.Errorf("combatant %q: initiative is required", name)
		}
		initiative, err := strconv.Atoi(initiativeText)
		if err != nil {
			return nil, fmt.Errorf("combatant %q: invalid initiative %q", name, initiativeText)
		}
		levelText := strings.TrimSpace(row.level.Text)
		if levelText == "" {
			return nil, fmt.Errorf("combatant %q: level is required", name)
		}
		level, err := strconv.Atoi(levelText)
		if err != nil || level < 1 {
			return nil, fmt.Errorf("combatant %q: invalid level %q", name, levelText)
		}
		countText := strings.TrimSpace(row.number.Text)
		if countText == "" {
			countText = "1"
		}
		count, err := strconv.Atoi(countText)
		if err != nil || count < 1 {
			return nil, fmt.Errorf("combatant %q: invalid number %q", name, countText)
		}
		xpText := strings.TrimSpace(row.xp.Text)
		if xpText == "" {
			return nil, fmt.Errorf("combatant %q: XP is required", name)
		}
		xp, err := strconv.Atoi(xpText)
		if err != nil || xp < 0 {
			return nil, fmt.Errorf("combatant %q: invalid XP %q", name, xpText)
		}
		hpText := strings.TrimSpace(row.hp.Text)
		if hpText == "" {
			return nil, fmt.Errorf("combatant %q: HP is required", name)
		}
		hp, err := strconv.Atoi(hpText)
		if err != nil || hp < 0 {
			return nil, fmt.Errorf("combatant %q: invalid HP %q", name, hpText)
		}
		hpMaxText := strings.TrimSpace(row.hpMax.Text)
		if hpMaxText == "" {
			return nil, fmt.Errorf("combatant %q: max HP is required", name)
		}
		hpMax, err := strconv.Atoi(hpMaxText)
		if err != nil || hpMax < 1 {
			return nil, fmt.Errorf("combatant %q: invalid max HP %q", name, hpMaxText)
		}
		if hp > hpMax {
			return nil, fmt.Errorf("combatant %q: current HP cannot exceed max HP", name)
		}
		defenseText := strings.TrimSpace(row.defense.Text)
		if defenseText == "" {
			return nil, fmt.Errorf("combatant %q: defense is required", name)
		}
		defense, err := strconv.Atoi(defenseText)
		if err != nil || defense < 0 {
			return nil, fmt.Errorf("combatant %q: invalid defense %q", name, defenseText)
		}
		torsoOnly := row.torsoOnly != nil && row.torsoOnly.Checked
		resistance, err := row.resistance.collectProfile(name, torsoOnly)
		if err != nil {
			return nil, err
		}

		side := domain.SideNPC
		if row.side.Selected == "party" {
			side = domain.SideParty
			xp = 0
			count = 1
			if strings.TrimSpace(row.playerCharacterID) == "" {
				return nil, fmt.Errorf("combatant %q: party members must be loaded from campaign", name)
			}
		}

		for i := 0; i < count; i++ {
			combatant := domain.Combatant{
				ID:                strings.TrimSpace(row.combatantID),
				PlayerCharacterID: strings.TrimSpace(row.playerCharacterID),
				Name:              name,
				Side:              side,
				Level:             level,
				XP:                xp,
				Initiative:        initiative,
				HP:                hp,
				MaxHP:             hpMax,
				Defense:           defense,
				TorsoOnly:         torsoOnly,
			}
			combatant.SetResistanceProfile(resistance)
			combatants = append(combatants, combatant)
		}
	}

	if len(combatants) == 0 {
		return nil, fmt.Errorf("add at least one combatant")
	}

	return combatants, nil
}

func collectCampaignPlayersFromRows(rows []*campaignPlayerInputRow) ([]domain.NewCampaignPlayer, error) {
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, row := range rows {
		playerName := strings.TrimSpace(row.playerName.Text)
		characterName := strings.TrimSpace(row.characterName.Text)
		if playerName == "" && characterName == "" {
			continue
		}
		if playerName == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if characterName == "" {
			return nil, fmt.Errorf("character name is required for player %q", playerName)
		}

		level, err := parsePositiveIntOrError(strings.TrimSpace(row.level.Text), "level", playerName)
		if err != nil {
			return nil, err
		}
		initiative, err := parseNonNegativeIntOrError(strings.TrimSpace(row.initiative.Text), "initiative", playerName)
		if err != nil {
			return nil, err
		}
		hp, err := parseNonNegativeIntOrError(strings.TrimSpace(row.hp.Text), "HP", playerName)
		if err != nil {
			return nil, err
		}
		hpMax, err := parsePositiveIntOrError(strings.TrimSpace(row.hpMax.Text), "max HP", playerName)
		if err != nil {
			return nil, err
		}
		if hp > hpMax {
			return nil, fmt.Errorf("current HP cannot exceed max HP for %q", playerName)
		}
		defense, err := parseNonNegativeIntOrError(strings.TrimSpace(row.defense.Text), "defense", playerName)
		if err != nil {
			return nil, err
		}
		resistance, err := row.resistance.collectProfile(characterName, false)
		if err != nil {
			return nil, err
		}
		special := domain.SpecialValues{}
		for _, attribute := range domain.SpecialAttributes() {
			value, err := parsePositiveIntOrError(strings.TrimSpace(row.special[attribute].Text), string(attribute), playerName)
			if err != nil {
				return nil, err
			}
			if err := special.Set(attribute, value); err != nil {
				return nil, err
			}
		}

		character := domain.Combatant{
			Name:       characterName,
			Side:       domain.SideParty,
			Level:      level,
			Initiative: initiative,
			HP:         hp,
			MaxHP:      hpMax,
			Defense:    defense,
		}
		character.SetResistanceProfile(resistance)
		players = append(players, domain.NewCampaignPlayer{
			PlayerName: playerName,
			Notes:      row.notes.Text,
			Special:    special,
			Inactive:   row.active != nil && !row.active.Checked,
			Character:  character,
		})
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	return players, nil
}
