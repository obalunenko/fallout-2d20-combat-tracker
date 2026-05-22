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
		drEnergyHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyHead.Text), "DR energy head", name)
		if err != nil {
			return nil, err
		}
		drEnergyTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyTorso.Text), "DR energy torso", name)
		if err != nil {
			return nil, err
		}
		drEnergyLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyLA.Text), "DR energy left arm", name)
		if err != nil {
			return nil, err
		}
		drEnergyRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyRA.Text), "DR energy right arm", name)
		if err != nil {
			return nil, err
		}
		drEnergyLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyLL.Text), "DR energy left leg", name)
		if err != nil {
			return nil, err
		}
		drEnergyRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyRL.Text), "DR energy right leg", name)
		if err != nil {
			return nil, err
		}
		drRadHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadHead.Text), "DR radiation head", name)
		if err != nil {
			return nil, err
		}
		drRadTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadTorso.Text), "DR radiation torso", name)
		if err != nil {
			return nil, err
		}
		drRadLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadLA.Text), "DR radiation left arm", name)
		if err != nil {
			return nil, err
		}
		drRadRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadRA.Text), "DR radiation right arm", name)
		if err != nil {
			return nil, err
		}
		drRadLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadLL.Text), "DR radiation left leg", name)
		if err != nil {
			return nil, err
		}
		drRadRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadRL.Text), "DR radiation right leg", name)
		if err != nil {
			return nil, err
		}
		drPhysHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysHead.Text), "DR physical head", name)
		if err != nil {
			return nil, err
		}
		drPhysTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysTorso.Text), "DR physical torso", name)
		if err != nil {
			return nil, err
		}
		drPhysLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysLA.Text), "DR physical left arm", name)
		if err != nil {
			return nil, err
		}
		drPhysRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysRA.Text), "DR physical right arm", name)
		if err != nil {
			return nil, err
		}
		drPhysLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysLL.Text), "DR physical left leg", name)
		if err != nil {
			return nil, err
		}
		drPhysRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysRL.Text), "DR physical right leg", name)
		if err != nil {
			return nil, err
		}
		drPoison, immPoison, err := parseResistanceCell(name, "poison", row.drPoison.Text, row.immPoison.Checked)
		if err != nil {
			return nil, err
		}
		if row.torsoOnly != nil && row.torsoOnly.Checked {
			drPhysHead, drPhysLA, drPhysRA, drPhysLL, drPhysRL = 0, 0, 0, 0, 0
			drEnergyHead, drEnergyLA, drEnergyRA, drEnergyLL, drEnergyRL = 0, 0, 0, 0, 0
			drRadHead, drRadLA, drRadRA, drRadLL, drRadRL = 0, 0, 0, 0, 0
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
			combatants = append(combatants, domain.Combatant{
				ID:                      strings.TrimSpace(row.combatantID),
				PlayerCharacterID:       strings.TrimSpace(row.playerCharacterID),
				Name:                    name,
				Side:                    side,
				Level:                   level,
				XP:                      xp,
				Initiative:              initiative,
				HP:                      hp,
				MaxHP:                   hpMax,
				Defense:                 defense,
				TorsoOnly:               row.torsoOnly != nil && row.torsoOnly.Checked,
				ResistPhysicalHead:      drPhysHead,
				ResistPhysicalTorso:     drPhysTorso,
				ResistPhysicalLeftArm:   drPhysLA,
				ResistPhysicalRightArm:  drPhysRA,
				ResistPhysicalLeftLeg:   drPhysLL,
				ResistPhysicalRightLeg:  drPhysRL,
				ResistEnergyHead:        drEnergyHead,
				ResistEnergyTorso:       drEnergyTorso,
				ResistEnergyLeftArm:     drEnergyLA,
				ResistEnergyRightArm:    drEnergyRA,
				ResistEnergyLeftLeg:     drEnergyLL,
				ResistEnergyRightLeg:    drEnergyRL,
				ResistRadiationHead:     drRadHead,
				ResistRadiationTorso:    drRadTorso,
				ResistRadiationLeftArm:  drRadLA,
				ResistRadiationRightArm: drRadRA,
				ResistRadiationLeftLeg:  drRadLL,
				ResistRadiationRightLeg: drRadRL,
				ImmunePhysical:          row.immPhysical.Checked,
				ImmuneEnergy:            row.immEnergy.Checked,
				ImmuneRadiation:         row.immRadiation.Checked,
				ResistPoison:            drPoison,
				ImmunePoison:            immPoison,
			})
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
		drEnergyHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyHead.Text), "DR energy head", playerName)
		if err != nil {
			return nil, err
		}
		drEnergyTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyTorso.Text), "DR energy torso", playerName)
		if err != nil {
			return nil, err
		}
		drEnergyLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyLA.Text), "DR energy left arm", playerName)
		if err != nil {
			return nil, err
		}
		drEnergyRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyRA.Text), "DR energy right arm", playerName)
		if err != nil {
			return nil, err
		}
		drEnergyLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyLL.Text), "DR energy left leg", playerName)
		if err != nil {
			return nil, err
		}
		drEnergyRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drEnergyRL.Text), "DR energy right leg", playerName)
		if err != nil {
			return nil, err
		}
		drRadHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadHead.Text), "DR radiation head", playerName)
		if err != nil {
			return nil, err
		}
		drRadTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadTorso.Text), "DR radiation torso", playerName)
		if err != nil {
			return nil, err
		}
		drRadLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadLA.Text), "DR radiation left arm", playerName)
		if err != nil {
			return nil, err
		}
		drRadRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadRA.Text), "DR radiation right arm", playerName)
		if err != nil {
			return nil, err
		}
		drRadLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadLL.Text), "DR radiation left leg", playerName)
		if err != nil {
			return nil, err
		}
		drRadRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drRadRL.Text), "DR radiation right leg", playerName)
		if err != nil {
			return nil, err
		}
		drPhysHead, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysHead.Text), "DR physical head", playerName)
		if err != nil {
			return nil, err
		}
		drPhysTorso, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysTorso.Text), "DR physical torso", playerName)
		if err != nil {
			return nil, err
		}
		drPhysLA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysLA.Text), "DR physical left arm", playerName)
		if err != nil {
			return nil, err
		}
		drPhysRA, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysRA.Text), "DR physical right arm", playerName)
		if err != nil {
			return nil, err
		}
		drPhysLL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysLL.Text), "DR physical left leg", playerName)
		if err != nil {
			return nil, err
		}
		drPhysRL, err := parseNonNegativeIntOrError(strings.TrimSpace(row.drPhysRL.Text), "DR physical right leg", playerName)
		if err != nil {
			return nil, err
		}
		drPoison, immPoison, err := parseResistanceCell(characterName, "poison", row.drPoison.Text, row.immPoison.Checked)
		if err != nil {
			return nil, err
		}

		players = append(players, domain.NewCampaignPlayer{
			PlayerName: playerName,
			Character: domain.Combatant{
				Name:                    characterName,
				Side:                    domain.SideParty,
				Level:                   level,
				Initiative:              initiative,
				HP:                      hp,
				MaxHP:                   hpMax,
				Defense:                 defense,
				ResistPhysicalHead:      drPhysHead,
				ResistPhysicalTorso:     drPhysTorso,
				ResistPhysicalLeftArm:   drPhysLA,
				ResistPhysicalRightArm:  drPhysRA,
				ResistPhysicalLeftLeg:   drPhysLL,
				ResistPhysicalRightLeg:  drPhysRL,
				ResistEnergyHead:        drEnergyHead,
				ResistEnergyTorso:       drEnergyTorso,
				ResistEnergyLeftArm:     drEnergyLA,
				ResistEnergyRightArm:    drEnergyRA,
				ResistEnergyLeftLeg:     drEnergyLL,
				ResistEnergyRightLeg:    drEnergyRL,
				ResistRadiationHead:     drRadHead,
				ResistRadiationTorso:    drRadTorso,
				ResistRadiationLeftArm:  drRadLA,
				ResistRadiationRightArm: drRadRA,
				ResistRadiationLeftLeg:  drRadLL,
				ResistRadiationRightLeg: drRadRL,
				ImmunePhysical:          row.immPhysical.Checked,
				ImmuneEnergy:            row.immEnergy.Checked,
				ImmuneRadiation:         row.immRadiation.Checked,
				ResistPoison:            drPoison,
				ImmunePoison:            immPoison,
			},
		})
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	return players, nil
}
