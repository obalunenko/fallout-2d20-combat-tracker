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
	name          *widget.Entry
	side          *widget.Select
	torsoOnly     *widget.Check
	number        *widget.Entry
	level         *widget.Entry
	xp            *widget.Entry
	initiative    *widget.Entry
	hp            *widget.Entry
	hpMax         *widget.Entry
	defense       *widget.Entry
	drEnergyHead  *widget.Entry
	drEnergyTorso *widget.Entry
	drEnergyLA    *widget.Entry
	drEnergyRA    *widget.Entry
	drEnergyLL    *widget.Entry
	drEnergyRL    *widget.Entry
	drRadHead     *widget.Entry
	drRadTorso    *widget.Entry
	drRadLA       *widget.Entry
	drRadRA       *widget.Entry
	drRadLL       *widget.Entry
	drRadRL       *widget.Entry
	drPhysHead    *widget.Entry
	drPhysTorso   *widget.Entry
	drPhysLA      *widget.Entry
	drPhysRA      *widget.Entry
	drPhysLL      *widget.Entry
	drPhysRL      *widget.Entry
	drPoison      *widget.Entry
	immPhysical   *widget.Check
	immEnergy     *widget.Check
	immRadiation  *widget.Check
	immPoison     *widget.Check
	root          *fyne.Container
}

type campaignPlayerInputRow struct {
	playerName    *widget.Entry
	characterName *widget.Entry
	level         *widget.Entry
	initiative    *widget.Entry
	hp            *widget.Entry
	hpMax         *widget.Entry
	defense       *widget.Entry
	drEnergyHead  *widget.Entry
	drEnergyTorso *widget.Entry
	drEnergyLA    *widget.Entry
	drEnergyRA    *widget.Entry
	drEnergyLL    *widget.Entry
	drEnergyRL    *widget.Entry
	drRadHead     *widget.Entry
	drRadTorso    *widget.Entry
	drRadLA       *widget.Entry
	drRadRA       *widget.Entry
	drRadLL       *widget.Entry
	drRadRL       *widget.Entry
	drPhysHead    *widget.Entry
	drPhysTorso   *widget.Entry
	drPhysLA      *widget.Entry
	drPhysRA      *widget.Entry
	drPhysLL      *widget.Entry
	drPhysRL      *widget.Entry
	drPoison      *widget.Entry
	immPhysical   *widget.Check
	immEnergy     *widget.Check
	immRadiation  *widget.Check
	immPoison     *widget.Check
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
		strings.TrimSpace(row.drEnergyHead.Text) == "0" &&
		strings.TrimSpace(row.drEnergyTorso.Text) == "0" &&
		strings.TrimSpace(row.drEnergyLA.Text) == "0" &&
		strings.TrimSpace(row.drEnergyRA.Text) == "0" &&
		strings.TrimSpace(row.drEnergyLL.Text) == "0" &&
		strings.TrimSpace(row.drEnergyRL.Text) == "0" &&
		strings.TrimSpace(row.drRadHead.Text) == "0" &&
		strings.TrimSpace(row.drRadTorso.Text) == "0" &&
		strings.TrimSpace(row.drRadLA.Text) == "0" &&
		strings.TrimSpace(row.drRadRA.Text) == "0" &&
		strings.TrimSpace(row.drRadLL.Text) == "0" &&
		strings.TrimSpace(row.drRadRL.Text) == "0" &&
		strings.TrimSpace(row.drPhysHead.Text) == "0" &&
		strings.TrimSpace(row.drPhysTorso.Text) == "0" &&
		strings.TrimSpace(row.drPhysLA.Text) == "0" &&
		strings.TrimSpace(row.drPhysRA.Text) == "0" &&
		strings.TrimSpace(row.drPhysLL.Text) == "0" &&
		strings.TrimSpace(row.drPhysRL.Text) == "0" &&
		!row.immPhysical.Checked &&
		strings.TrimSpace(row.drPoison.Text) == "0" &&
		!row.immPoison.Checked
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
	row.torsoOnly.SetChecked(template.TorsoOnly)

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
	row.drEnergyHead.SetText(strconv.Itoa(template.ResistEnergyHead))
	row.drEnergyTorso.SetText(strconv.Itoa(template.ResistEnergyTorso))
	row.drEnergyLA.SetText(strconv.Itoa(template.ResistEnergyLeftArm))
	row.drEnergyRA.SetText(strconv.Itoa(template.ResistEnergyRightArm))
	row.drEnergyLL.SetText(strconv.Itoa(template.ResistEnergyLeftLeg))
	row.drEnergyRL.SetText(strconv.Itoa(template.ResistEnergyRightLeg))
	row.drRadHead.SetText(strconv.Itoa(template.ResistRadiationHead))
	row.drRadTorso.SetText(strconv.Itoa(template.ResistRadiationTorso))
	row.drRadLA.SetText(strconv.Itoa(template.ResistRadiationLeftArm))
	row.drRadRA.SetText(strconv.Itoa(template.ResistRadiationRightArm))
	row.drRadLL.SetText(strconv.Itoa(template.ResistRadiationLeftLeg))
	row.drRadRL.SetText(strconv.Itoa(template.ResistRadiationRightLeg))
	row.immPhysical.SetChecked(template.ImmunePhysical)
	if !template.ImmunePhysical {
		row.drPhysHead.SetText(strconv.Itoa(template.ResistPhysicalHead))
		row.drPhysTorso.SetText(strconv.Itoa(template.ResistPhysicalTorso))
		row.drPhysLA.SetText(strconv.Itoa(template.ResistPhysicalLeftArm))
		row.drPhysRA.SetText(strconv.Itoa(template.ResistPhysicalRightArm))
		row.drPhysLL.SetText(strconv.Itoa(template.ResistPhysicalLeftLeg))
		row.drPhysRL.SetText(strconv.Itoa(template.ResistPhysicalRightLeg))
	}
	row.immEnergy.SetChecked(template.ImmuneEnergy)
	row.immRadiation.SetChecked(template.ImmuneRadiation)

	row.immPoison.SetChecked(template.ImmunePoison)
	if !template.ImmunePoison {
		row.drPoison.SetText(strconv.Itoa(template.ResistPoison))
	}
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
