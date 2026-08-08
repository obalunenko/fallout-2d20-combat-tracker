package fyneui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
	label.SetText(formatActiveTargetDetails(enc, c))
}

func formatDRValue(value int, immune bool) string {
	if immune {
		return "IMM"
	}
	return strconv.Itoa(value)
}

func formatBodyResistanceTable(c domain.Combatant) string {
	damageTypes := domain.LocationDamageTypes()
	locations := domain.BodyLocations()

	locationHeader := "Location"
	locationWidth := textWidth(locationHeader)
	for _, location := range locations {
		locationWidth = max(locationWidth, textWidth(bodyLocationTitle(location)))
	}

	damageHeaders := make([]string, len(damageTypes))
	damageWidths := make([]int, len(damageTypes))
	rows := make([][]string, len(locations))
	for i, damageType := range damageTypes {
		damageHeaders[i] = damageTitleLabel(damageType)
		damageWidths[i] = textWidth(damageHeaders[i])
	}
	for locationIdx, location := range locations {
		rows[locationIdx] = make([]string, len(damageTypes))
		for damageIdx, damageType := range damageTypes {
			value := formatCombatantLocationResistance(c, damageType, location)
			rows[locationIdx][damageIdx] = value
			damageWidths[damageIdx] = max(damageWidths[damageIdx], textWidth(value))
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Body Damage Resistance\n%-*s", locationWidth, locationHeader)
	for i, header := range damageHeaders {
		fmt.Fprintf(&b, " | %*s", damageWidths[i], header)
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", resistanceTableWidth(locationWidth, damageWidths)))
	for locationIdx, location := range locations {
		fmt.Fprintf(&b, "\n%-*s", locationWidth, bodyLocationTitle(location))
		for damageIdx, value := range rows[locationIdx] {
			fmt.Fprintf(&b, " | %*s", damageWidths[damageIdx], value)
		}
	}
	return b.String()
}

func formatActiveTargetDetails(enc *domain.Encounter, c domain.Combatant) string {
	rows := [][2]string{
		{"Name", encounterDisplayNameByID(enc, c.ID)},
		{"Side", string(c.Side)},
		{"Level", strconv.Itoa(c.Level)},
		{"XP", strconv.Itoa(c.XP)},
		{"Initiative", strconv.Itoa(c.Initiative)},
		{"HP", fmt.Sprintf("%d/%d", c.HP, c.MaxHP)},
		{"Defense", strconv.Itoa(c.Defense)},
		{"Status", formatCombatantStatus(c)},
		{"DR Poison", formatCombatantGlobalResistance(c, domain.DamagePoison)},
	}
	if isTorsoOnlyCombatant(c) {
		rows = append(rows,
			[2]string{"Torso-only", "yes"},
			[2]string{"DR Physical", formatCombatantLocationResistance(c, domain.DamagePhysical, domain.BodyTorso)},
			[2]string{"DR Energy", formatCombatantLocationResistance(c, domain.DamageEnergy, domain.BodyTorso)},
			[2]string{"DR Radiation", formatCombatantLocationResistance(c, domain.DamageRadiation, domain.BodyTorso)},
		)
		return formatKeyValueTable("Participant Details", rows)
	}
	return formatKeyValueTable("Participant Details", rows) + "\n\n" + formatBodyResistanceTable(c)
}

func formatKeyValueTable(title string, rows [][2]string) string {
	keyHeader := "Field"
	valueHeader := "Value"
	keyWidth := textWidth(keyHeader)
	valueWidth := textWidth(valueHeader)
	for _, row := range rows {
		keyWidth = max(keyWidth, textWidth(row[0]))
		valueWidth = max(valueWidth, textWidth(row[1]))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%-*s | %-*s\n", title, keyWidth, keyHeader, valueWidth, valueHeader)
	b.WriteString(strings.Repeat("-", keyWidth+3+valueWidth))
	for _, row := range rows {
		fmt.Fprintf(&b, "\n%-*s | %-*s", keyWidth, row[0], valueWidth, row[1])
	}
	return b.String()
}

func formatCombatantStatus(c domain.Combatant) string {
	if isCombatantDefeated(c) {
		return "Defeated"
	}
	status := "Ready"
	if c.Active {
		status = "Active"
	}
	if healthState := combatantHealthState(c); healthState != "" {
		return status + ", " + healthState
	}
	return status
}

func isCombatantDefeated(c domain.Combatant) bool {
	return c.Defeated || c.HP <= 0
}

func combatantHealthState(c domain.Combatant) string {
	maxHP := c.MaxHP
	if maxHP <= 0 {
		maxHP = c.HP
	}
	if maxHP <= 0 || c.HP >= maxHP {
		return ""
	}
	if c.HP*4 <= maxHP {
		return "Critical"
	}
	if c.HP*2 <= maxHP {
		return "Wounded"
	}
	return ""
}

func combatantNeedsAttention(c domain.Combatant) bool {
	return combatantHealthState(c) != ""
}

func resistanceTableWidth(locationWidth int, damageWidths []int) int {
	width := locationWidth
	for _, damageWidth := range damageWidths {
		width += 3 + damageWidth
	}
	return width
}

func textWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func formatCombatantLine(enc *domain.Encounter, c domain.Combatant) string {
	name := c.Name
	if enc != nil {
		name = encounterDisplayNameByID(enc, c.ID)
	}
	prefix := "   "
	isDefeated := isCombatantDefeated(c)
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
	if healthState := combatantHealthState(c); healthState != "" {
		return line + " [" + strings.ToUpper(healthState) + "]"
	}
	return line
}

func formatExpandedCombatantDetails(enc *domain.Encounter, c domain.Combatant) string {
	return formatActiveTargetDetails(enc, c)
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
	if s.Difficulty == string(domain.EncounterDifficultyUnknown) {
		if strings.TrimSpace(s.DifficultyUnavailableReason) == "" {
			return s.Difficulty
		}
		return fmt.Sprintf("%s (%s)", s.Difficulty, s.DifficultyUnavailableReason)
	}
	return fmt.Sprintf(
		"%s (party:%d avg PC lvl:%d | monster XP:%d baseline:%.1f | encounter lvl:%d diff:%+d)",
		s.Difficulty,
		s.PartyCount,
		s.AveragePCLevel,
		s.TotalMonsterXP,
		s.XPBaseline,
		s.EncounterLevel,
		s.DifficultyDifference,
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
	if metrics.Label == domain.EncounterDifficultyUnknown {
		reason := strings.TrimSpace(metrics.UnavailableReason)
		if reason == "" {
			reason = "difficulty inputs are unavailable"
		}
		return fmt.Sprintf("Difficulty: Unknown (%s)", reason)
	}
	return fmt.Sprintf(
		"Difficulty: %s (party: %d avg PC lvl %d | monster XP: %d baseline %.1f | encounter lvl %d diff %+d)",
		metrics.Label,
		metrics.PartyCount,
		metrics.AveragePCLevel,
		metrics.TotalMonsterXP,
		metrics.XPBaseline,
		metrics.EncounterLevel,
		metrics.Difference,
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
