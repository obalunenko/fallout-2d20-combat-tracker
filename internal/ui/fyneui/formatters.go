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
				"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nTorso-only: yes\n%s\nStatus: %s",
				displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP,
				c.Defense,
				formatTorsoResistanceLines(c),
				status,
			),
		)
		return
	}
	label.SetText(
		fmt.Sprintf(
			"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\n%s\nStatus: %s",
			displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP,
			c.Defense,
			formatCompactBodyResistanceLines(c),
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

func formatTorsoResistanceLines(c domain.Combatant) string {
	var b strings.Builder
	for _, damageType := range domain.LocationDamageTypes() {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(
			&b,
			"DR %s: %s",
			damageTitleLabel(damageType),
			formatCombatantLocationResistance(c, damageType, domain.BodyTorso),
		)
	}
	fmt.Fprintf(&b, "\nDR Poison: %s", formatCombatantGlobalResistance(c, domain.DamagePoison))
	return b.String()
}

func formatCompactBodyResistanceLines(c domain.Combatant) string {
	var b strings.Builder
	for _, damageType := range domain.LocationDamageTypes() {
		for _, location := range domain.BodyLocations() {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(
				&b,
				"%s %s: %s",
				damageAbbreviation(damageType),
				bodyLocationTitle(location),
				formatCombatantLocationResistance(c, damageType, location),
			)
		}
	}
	fmt.Fprintf(&b, "\nDR Poison: %s", formatCombatantGlobalResistance(c, domain.DamagePoison))
	return b.String()
}

func formatBodyResistanceTable(c domain.Combatant) string {
	damageTypes := domain.LocationDamageTypes()
	var b strings.Builder
	b.WriteString("Body Damage Resistance\nLocation ")
	for _, damageType := range damageTypes {
		fmt.Fprintf(&b, " | %9s", damageTitleLabel(damageType))
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 9+12*len(damageTypes)))
	for _, location := range domain.BodyLocations() {
		fmt.Fprintf(&b, "\n%-8s", bodyLocationTitle(location))
		for _, damageType := range damageTypes {
			fmt.Fprintf(&b, " | %9s", formatCombatantLocationResistance(c, damageType, location))
		}
	}
	return b.String()
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
		prefix, name, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP, c.Defense, formatCombatantGlobalResistance(c, domain.DamagePoison),
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
			"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\n%s",
			encounterDisplayNameByID(enc, c.ID),
			c.Side,
			c.Level,
			c.XP,
			c.Initiative,
			c.HP,
			c.MaxHP,
			c.Defense,
			status,
			formatTorsoResistanceLines(c),
		)
	}
	return fmt.Sprintf(
		"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\nDR Poison: %s\n\n%s",
		encounterDisplayNameByID(enc, c.ID),
		c.Side,
		c.Level,
		c.XP,
		c.Initiative,
		c.HP,
		c.MaxHP,
		c.Defense,
		status,
		formatCombatantGlobalResistance(c, domain.DamagePoison),
		formatBodyResistanceTable(c),
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
		return "No players in campaign"
	}
	lines := make([]string, 0, len(players))
	for i, p := range players {
		status := "active"
		if p.Inactive {
			status = "inactive"
		}
		lines = append(lines, fmt.Sprintf(
			"[%02d] %s -> %s | %s | Lvl:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
			i+1,
			p.PlayerName,
			p.Character.Name,
			status,
			p.Character.Level,
			p.Character.Initiative,
			p.Character.HP,
			p.Character.MaxHP,
			p.Character.Defense,
			formatCombatantGlobalResistance(p.Character, domain.DamagePoison),
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
			formatCombatantGlobalResistance(c, domain.DamagePoison),
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
