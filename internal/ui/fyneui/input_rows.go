package fyneui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type combatantInputRow struct {
	combatantID       string
	playerCharacterID string
	linkedParty       bool
	name              *widget.Entry
	side              *widget.Select
	torsoOnly         *widget.Check
	number            *widget.Entry
	level             *widget.Entry
	xp                *widget.Entry
	initiative        *widget.Entry
	hp                *widget.Entry
	hpMax             *widget.Entry
	defense           *widget.Entry
	resistance        uiResistanceInputs
	root              *fyne.Container
}

type campaignPlayerInputRow struct {
	playerName    *widget.Entry
	characterName *widget.Entry
	level         *widget.Entry
	initiative    *widget.Entry
	hp            *widget.Entry
	hpMax         *widget.Entry
	defense       *widget.Entry
	resistance    uiResistanceInputs
	active        *widget.Check
	root          *fyne.Container
}

func newResistanceInputCell(entry *widget.Entry, onChanged func()) (fyne.CanvasObject, *widget.Check) {
	immune := widget.NewCheck("immune", func(checked bool) {
		if checked {
			entry.SetText("0")
			entry.Disable()
			if onChanged != nil {
				onChanged()
			}
			return
		}
		entry.Enable()
		if onChanged != nil {
			onChanged()
		}
	})
	return container.NewBorder(nil, nil, nil, immune, entry), immune
}

func newGlobalImmunityCheck(label string, entries []*widget.Entry, onChanged func()) *widget.Check {
	immune := widget.NewCheck(label, func(checked bool) {
		for _, entry := range entries {
			if checked {
				entry.SetText("0")
				entry.Disable()
				continue
			}
			entry.Enable()
		}
		if onChanged != nil {
			onChanged()
		}
	})
	return immune
}

func newTableHeaderLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	return l
}

func combatantInputRowIsEmpty(row *combatantInputRow) bool {
	if row == nil {
		return true
	}
	return strings.TrimSpace(row.name.Text) == "" &&
		strings.TrimSpace(row.initiative.Text) == "" &&
		strings.TrimSpace(row.level.Text) == "1" &&
		strings.TrimSpace(row.xp.Text) == "0" &&
		strings.TrimSpace(row.hp.Text) == "1" &&
		strings.TrimSpace(row.hpMax.Text) == "1" &&
		strings.TrimSpace(row.defense.Text) == "0" &&
		row.resistance.isZero()
}

func fillCombatantInputRow(row *combatantInputRow, template domain.Combatant, side domain.Side, count int) {
	if row == nil {
		return
	}
	if count < 1 {
		count = 1
	}

	selectedSide := string(side)
	if selectedSide != "npc" {
		selectedSide = "party"
	}
	row.side.SetSelected(selectedSide)
	row.combatantID = strings.TrimSpace(template.ID)
	row.playerCharacterID = strings.TrimSpace(template.PlayerCharacterID)
	if row.playerCharacterID == "" && side == domain.SideParty {
		row.playerCharacterID = row.combatantID
		row.combatantID = ""
	}

	row.name.SetText(strings.TrimSpace(template.Name))
	row.number.SetText(strconv.Itoa(count))
	row.level.SetText(strconv.Itoa(template.Level))
	row.xp.SetText(strconv.Itoa(template.XP))
	row.initiative.SetText(strconv.Itoa(template.Initiative))
	row.hp.SetText(strconv.Itoa(template.HP))
	maxHP := template.MaxHP
	if maxHP <= 0 {
		maxHP = template.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	row.hpMax.SetText(strconv.Itoa(maxHP))
	row.defense.SetText(strconv.Itoa(template.Defense))
	row.torsoOnly.SetChecked(template.TorsoOnly)
	row.resistance.setProfile(template.ResistanceProfile())
	setCombatantInputRowLinkedParty(row, side == domain.SideParty && row.playerCharacterID != "")
}

func resetCombatantInputRow(row *combatantInputRow, side domain.Side) {
	if row == nil {
		return
	}
	setCombatantInputRowLinkedParty(row, false)
	row.combatantID = ""
	row.playerCharacterID = ""
	row.name.SetText("")
	row.number.SetText("1")
	row.level.SetText("1")
	row.xp.SetText("0")
	row.initiative.SetText("")
	row.hp.SetText("1")
	row.hpMax.SetText("1")
	row.defense.SetText("0")
	row.resistance.reset()
	if side != domain.SideParty {
		setCombatantSideOptions(row, []string{"npc"})
		row.side.SetSelected("npc")
		row.torsoOnly.SetChecked(true)
		return
	}
	setCombatantSideOptions(row, []string{"party"})
	row.side.SetSelected("party")
	row.torsoOnly.SetChecked(false)
}

func setCombatantInputRowLinkedParty(row *combatantInputRow, locked bool) {
	if row == nil {
		return
	}
	wasLinked := row.linkedParty
	row.linkedParty = locked
	if locked {
		setCombatantSideOptions(row, []string{"party"})
		row.side.SetSelected("party")
		disableCombatantInputRowStats(row)
		return
	}
	if row.side.Selected == "party" {
		setCombatantSideOptions(row, []string{"party"})
	} else {
		setCombatantSideOptions(row, []string{"npc"})
	}
	if wasLinked {
		enableCombatantInputRowStats(row)
	}
}

func setCombatantSideOptions(row *combatantInputRow, options []string) {
	row.side.Options = options
	row.side.Refresh()
}

func disableCombatantInputRowStats(row *combatantInputRow) {
	row.name.Disable()
	row.side.Disable()
	row.torsoOnly.Disable()
	row.number.Disable()
	row.level.Disable()
	row.xp.Disable()
	row.initiative.Disable()
	row.hp.Disable()
	row.hpMax.Disable()
	row.defense.Disable()
	for _, entry := range row.resistance.entries() {
		entry.Disable()
	}
	for _, damageType := range domain.DamageTypes() {
		if check := row.resistance.globalImmune(damageType); check != nil {
			check.Disable()
		}
	}
}

func enableCombatantInputRowStats(row *combatantInputRow) {
	row.name.Enable()
	row.side.Enable()
	row.torsoOnly.Enable()
	row.level.Enable()
	row.initiative.Enable()
	row.hp.Enable()
	row.hpMax.Enable()
	row.defense.Enable()
	for _, entry := range row.resistance.entries() {
		entry.Enable()
	}
	for _, damageType := range domain.DamageTypes() {
		if check := row.resistance.globalImmune(damageType); check != nil {
			check.Enable()
		}
	}
	if row.side.Selected == "party" {
		row.number.Disable()
		row.xp.Disable()
		return
	}
	row.number.Enable()
	row.xp.Enable()
}

func collectCombatantsPreviewFromRows(rows []*combatantInputRow) []domain.Combatant {
	preview := make([]domain.Combatant, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.name.Text)
		if name == "" {
			continue
		}

		side := domain.SideNPC
		if row.side.Selected == "party" {
			side = domain.SideParty
		}

		level := 1
		if parsed, err := strconv.Atoi(strings.TrimSpace(row.level.Text)); err == nil && parsed > 0 {
			level = parsed
		}

		count := 1
		if side == domain.SideNPC {
			if parsed, err := strconv.Atoi(strings.TrimSpace(row.number.Text)); err == nil && parsed > 0 {
				count = parsed
			}
		}

		xp := 0
		if side == domain.SideNPC {
			if parsed, err := strconv.Atoi(strings.TrimSpace(row.xp.Text)); err == nil && parsed >= 0 {
				xp = parsed
			}
		}

		for i := 0; i < count; i++ {
			preview = append(preview, domain.Combatant{
				Name:  name,
				Side:  side,
				Level: level,
				XP:    xp,
			})
		}
	}
	return preview
}
