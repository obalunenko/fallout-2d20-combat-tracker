package fyneui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/widget"
	"github.com/obalunenko/fallout/internal/domain"
)

func refreshSelected(label *widget.Label, enc *domain.Encounter, idx int) {
	if enc == nil || len(enc.Combatants) == 0 {
		label.SetText("No combatants")
		return
	}
	if idx < 0 || idx >= len(enc.Combatants) {
		idx = 0
	}
	c := enc.Combatants[idx]
	displayName := encounterDisplayNameByID(enc, c.ID)
	status := "Ready"
	if c.Defeated {
		status = "Defeated"
	}
	if isTorsoOnlyCombatant(c) {
		label.SetText(
			fmt.Sprintf(
				"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nTorso-only: yes\nDR Physical: %s\nDR Energy: %s\nDR Radiation: %s\nDR Poison: %s\nStatus: %s",
				displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP,
				c.Defense,
				formatDRValue(c.ResistPhysicalTorso, c.ImmunePhysical),
				formatDRValue(c.ResistEnergyTorso, c.ImmuneEnergy),
				formatDRValue(c.ResistRadiationTorso, c.ImmuneRadiation),
				formatDRValue(c.ResistPoison, c.ImmunePoison),
				status,
			),
		)
		return
	}
	label.SetText(
		fmt.Sprintf(
			"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nDEF Head: %d\nDEF Torso: %d\nDEF Left Arm: %d\nDEF Right Arm: %d\nDEF Left Leg: %d\nDEF Right Leg: %d\nDRP Head: %d\nDRP Torso: %d\nDRP Left Arm: %d\nDRP Right Arm: %d\nDRP Left Leg: %d\nDRP Right Leg: %d\nDRE Head: %d\nDRE Torso: %d\nDRE Left Arm: %d\nDRE Right Arm: %d\nDRE Left Leg: %d\nDRE Right Leg: %d\nDRR Head: %d\nDRR Torso: %d\nDRR Left Arm: %d\nDRR Right Arm: %d\nDRR Left Leg: %d\nDRR Right Leg: %d\nDR Poison: %s\nStatus: %s",
			displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP,
			c.Defense,
			c.DefenseHead, c.DefenseTorso, c.DefenseLeftArm, c.DefenseRightArm, c.DefenseLeftLeg, c.DefenseRightLeg,
			c.ResistPhysicalHead, c.ResistPhysicalTorso, c.ResistPhysicalLeftArm, c.ResistPhysicalRightArm, c.ResistPhysicalLeftLeg, c.ResistPhysicalRightLeg,
			c.ResistEnergyHead, c.ResistEnergyTorso, c.ResistEnergyLeftArm, c.ResistEnergyRightArm, c.ResistEnergyLeftLeg, c.ResistEnergyRightLeg,
			c.ResistRadiationHead, c.ResistRadiationTorso, c.ResistRadiationLeftArm, c.ResistRadiationRightArm, c.ResistRadiationLeftLeg, c.ResistRadiationRightLeg,
			formatDRValue(c.ResistPoison, c.ImmunePoison),
			status,
		),
	)
}

func formatDRValue(value int, immune bool) string {
	if immune {
		return "IMM"
	}
	return strconv.Itoa(value)
}

func formatCombatantLine(enc *domain.Encounter, c domain.Combatant) string {
	name := c.Name
	if enc != nil {
		name = encounterDisplayNameByID(enc, c.ID)
	}
	prefix := "   "
	isDefeated := c.Defeated || c.HP <= 0
	if c.Active && !isDefeated {
		prefix = ">> "
	} else if isDefeated {
		prefix = "xx "
	}
	line := fmt.Sprintf(
		"%s%s [%s] Lvl:%d XP:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
		prefix, name, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP, c.Defense, formatDRValue(c.ResistPoison, c.ImmunePoison),
	)
	if isDefeated {
		return line + " [DEFEATED]"
	}
	return line
}

func formatExpandedCombatantDetails(enc *domain.Encounter, c domain.Combatant) string {
	status := "Ready"
	if c.Defeated || c.HP <= 0 {
		status = "Defeated"
	} else if c.Active {
		status = "Active"
	}
	if isTorsoOnlyCombatant(c) {
		return fmt.Sprintf(
			"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\nDR Physical: %s\nDR Energy: %s\nDR Radiation: %s\nDR Poison: %s",
			encounterDisplayNameByID(enc, c.ID),
			c.Side,
			c.Level,
			c.XP,
			c.Initiative,
			c.HP,
			c.MaxHP,
			c.Defense,
			status,
			formatDRValue(c.ResistPhysicalTorso, c.ImmunePhysical),
			formatDRValue(c.ResistEnergyTorso, c.ImmuneEnergy),
			formatDRValue(c.ResistRadiationTorso, c.ImmuneRadiation),
			formatDRValue(c.ResistPoison, c.ImmunePoison),
		)
	}
	return fmt.Sprintf(
		"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\nDR Poison: %s\n\nBody Defense / Damage Resistance\nLocation  | Defense | Physical | Energy | Radiation\n-----------------------------------------------------\nHead      | %7d | %8d | %6d | %9d\nTorso     | %7d | %8d | %6d | %9d\nLeft Arm  | %7d | %8d | %6d | %9d\nRight Arm | %7d | %8d | %6d | %9d\nLeft Leg  | %7d | %8d | %6d | %9d\nRight Leg | %7d | %8d | %6d | %9d",
		encounterDisplayNameByID(enc, c.ID),
		c.Side,
		c.Level,
		c.XP,
		c.Initiative,
		c.HP,
		c.MaxHP,
		c.Defense,
		status,
		formatDRValue(c.ResistPoison, c.ImmunePoison),
		c.DefenseHead, c.ResistPhysicalHead, c.ResistEnergyHead, c.ResistRadiationHead,
		c.DefenseTorso, c.ResistPhysicalTorso, c.ResistEnergyTorso, c.ResistRadiationTorso,
		c.DefenseLeftArm, c.ResistPhysicalLeftArm, c.ResistEnergyLeftArm, c.ResistRadiationLeftArm,
		c.DefenseRightArm, c.ResistPhysicalRightArm, c.ResistEnergyRightArm, c.ResistRadiationRightArm,
		c.DefenseLeftLeg, c.ResistPhysicalLeftLeg, c.ResistEnergyLeftLeg, c.ResistRadiationLeftLeg,
		c.DefenseRightLeg, c.ResistPhysicalRightLeg, c.ResistEnergyRightLeg, c.ResistRadiationRightLeg,
	)
}

func formatCampaignStartDate(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(domain.CampaignDateLayout)
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func formatEncounterDifficultySummary(s domain.EncounterSummary) string {
	if strings.TrimSpace(s.Difficulty) == "" {
		return "Unknown"
	}
	if s.PartyCount == 0 || s.EnemyCount == 0 {
		return s.Difficulty
	}
	return fmt.Sprintf(
		"%s (P:%d avgLvl:%.1f budget:%d | NPC:%d avgLvl:%.1f XP:%d)",
		s.Difficulty,
		s.PartyCount,
		s.PartyAvgLevel,
		s.PartyXPBudget,
		s.EnemyCount,
		s.EnemyAvgLevel,
		s.EnemyTotalXP,
	)
}

func formatCampaignOverview(c *domain.Campaign) string {
	if c == nil {
		return "No active campaign"
	}
	return fmt.Sprintf(
		"Name: %s\nID: %s\nStart Date: %s\nUpdated: %s",
		c.Name,
		c.ID,
		formatCampaignStartDate(c.StartDate),
		formatTimestamp(c.UpdatedAt),
	)
}

func encounterDisplayNameByID(enc *domain.Encounter, combatantID string) string {
	if enc == nil {
		return ""
	}
	displayMap := encounterDisplayNameMap(enc)
	if name, ok := displayMap[combatantID]; ok {
		return name
	}
	for i := range enc.Combatants {
		if enc.Combatants[i].ID == combatantID {
			return enc.Combatants[i].Name
		}
	}
	return ""
}

func encounterDisplayNameMap(enc *domain.Encounter) map[string]string {
	names := make(map[string]string, len(enc.Combatants))
	npcCounts := make(map[string]int)
	for i := range enc.Combatants {
		c := enc.Combatants[i]
		if c.Side == domain.SideNPC {
			npcCounts[c.Name]++
		}
	}

	npcSeen := make(map[string]int)
	for i := range enc.Combatants {
		c := enc.Combatants[i]
		if c.Side == domain.SideNPC && npcCounts[c.Name] > 1 {
			npcSeen[c.Name]++
			names[c.ID] = fmt.Sprintf("%s (%s)", c.Name, alphabeticOrdinalLabel(npcSeen[c.Name]-1))
			continue
		}
		names[c.ID] = c.Name
	}
	return names
}

func alphabeticOrdinalLabel(idx int) string {
	if idx < 0 {
		return "A"
	}
	label := ""
	for idx >= 0 {
		label = string(rune('A'+(idx%26))) + label
		idx = idx/26 - 1
	}
	return label
}

func formatDifficultyPreview(metrics domain.EncounterDifficultyMetrics) string {
	if metrics.PartyCount == 0 || metrics.EnemyCount == 0 {
		return "Difficulty: Unknown (add at least one party member and one NPC)"
	}
	return fmt.Sprintf(
		"Difficulty: %s (xp ratio: %.2f | party: %d avg lvl %.1f budget %d | npc: %d avg lvl %.1f total xp: %d)",
		metrics.Label,
		metrics.Score,
		metrics.PartyCount,
		metrics.PartyAvgLevel,
		metrics.PartyXPBudget,
		metrics.EnemyCount,
		metrics.EnemyAvgLevel,
		metrics.EnemyTotalXP,
	)
}

func formatCampaignRoster(players []domain.NewCampaignPlayer) string {
	if len(players) == 0 {
		return "No active players in campaign"
	}
	lines := make([]string, 0, len(players))
	for i, p := range players {
		lines = append(lines, fmt.Sprintf(
			"[%02d] %s -> %s | Lvl:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
			i+1,
			p.PlayerName,
			p.Character.Name,
			p.Character.Level,
			p.Character.Initiative,
			p.Character.HP,
			p.Character.MaxHP,
			p.Character.Defense,
			formatDRValue(p.Character.ResistPoison, p.Character.ImmunePoison),
		))
	}
	return strings.Join(lines, "\n")
}

func formatPartyLibrary(members []domain.Combatant) string {
	if len(members) == 0 {
		return "No saved party members found in database"
	}
	lines := make([]string, 0, len(members))
	for i, c := range members {
		lines = append(lines, fmt.Sprintf(
			"[%02d] %s | Lvl:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
			i+1,
			c.Name,
			c.Level,
			c.Initiative,
			c.HP,
			c.MaxHP,
			c.Defense,
			formatDRValue(c.ResistPoison, c.ImmunePoison),
		))
	}
	return strings.Join(lines, "\n")
}

func formatTacticalSnapshot(encounter *domain.Encounter) string {
	if encounter == nil {
		return "No active encounter"
	}
	partyTotal, partyAlive, partyDefeated := 0, 0, 0
	npcTotal, npcAlive, npcDefeated := 0, 0, 0
	activeName := "-"
	for i := range encounter.Combatants {
		c := encounter.Combatants[i]
		isDefeated := c.Defeated || c.HP <= 0
		if c.Active && !isDefeated {
			activeName = encounterDisplayNameByID(encounter, c.ID)
		}
		if c.Side == domain.SideParty {
			partyTotal++
			if isDefeated {
				partyDefeated++
			} else {
				partyAlive++
			}
			continue
		}
		npcTotal++
		if isDefeated {
			npcDefeated++
		} else {
			npcAlive++
		}
	}
	return fmt.Sprintf(
		"Encounter: %s\nRound: %d\nActive Turn: %s\nParty: %d total / %d alive / %d defeated\nNPC: %d total / %d alive / %d defeated",
		encounter.Name,
		encounter.Round,
		activeName,
		partyTotal,
		partyAlive,
		partyDefeated,
		npcTotal,
		npcAlive,
		npcDefeated,
	)
}

func isTorsoOnlyCombatant(c domain.Combatant) bool {
	return c.TorsoOnly
}
