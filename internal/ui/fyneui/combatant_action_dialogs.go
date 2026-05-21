package fyneui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func showApplyDamageDialog(ctx context.Context, w fyne.Window, svc *appsvc.Service, enc *domain.Encounter, targetIndex int, refresh func()) {
	if enc == nil || len(enc.Combatants) == 0 {
		dialog.ShowError(fmt.Errorf("no combatants in active encounter"), w)
		return
	}
	if targetIndex < 0 || targetIndex >= len(enc.Combatants) {
		targetIndex = 0
	}

	target := enc.Combatants[targetIndex]
	targetDisplayName := encounterDisplayNameByID(enc, target.ID)
	typeSelect := widget.NewSelect([]string{"physical", "energy", "radiation", "poison"}, nil)
	typeSelect.SetSelected("physical")
	locationOptions := []string{"head", "torso", "left_arm", "right_arm", "left_leg", "right_leg"}
	if isTorsoOnlyCombatant(target) {
		locationOptions = []string{"torso"}
	}
	locationSelect := widget.NewSelect(locationOptions, nil)
	locationSelect.SetSelected("torso")

	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Damage amount")
	amountEntry.SetText("1")
	amountEntry.TextStyle = fyne.TextStyle{Monospace: true}

	damageDialog := dialog.NewForm(
		fmt.Sprintf("Apply Damage: %s", targetDisplayName),
		"Apply",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Type", typeSelect),
			widget.NewFormItem("Location", locationSelect),
			widget.NewFormItem("Amount", amountEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}

			damageType, err := parseDamageType(typeSelect.Selected)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			location, err := parseBodyLocation(locationSelect.Selected)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			amountText := strings.TrimSpace(amountEntry.Text)
			amount, err := strconv.Atoi(amountText)
			if err != nil || amount < 0 {
				dialog.ShowError(fmt.Errorf("invalid damage amount %q", amountText), w)
				return
			}

			_, _, err = svc.ExecuteApplyDamage(ctx, appsvc.ApplyDamageCommand{
				CombatantID: target.ID,
				DamageType:  damageType,
				Location:    location,
				Amount:      amount,
			})
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			if refresh != nil {
				refresh()
			}
		},
		w,
	)
	damageDialog.Resize(fyne.NewSize(420, 220))
	damageDialog.Show()
}

func showHealDialog(ctx context.Context, w fyne.Window, svc *appsvc.Service, enc *domain.Encounter, targetIndex int, refresh func()) {
	if enc == nil || len(enc.Combatants) == 0 {
		dialog.ShowError(fmt.Errorf("no combatants in active encounter"), w)
		return
	}
	if targetIndex < 0 || targetIndex >= len(enc.Combatants) {
		targetIndex = 0
	}

	target := enc.Combatants[targetIndex]
	targetDisplayName := encounterDisplayNameByID(enc, target.ID)
	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Heal amount")
	amountEntry.SetText("1")
	amountEntry.TextStyle = fyne.TextStyle{Monospace: true}

	healDialog := dialog.NewForm(
		fmt.Sprintf("Heal: %s", targetDisplayName),
		"Heal",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Amount", amountEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}

			amountText := strings.TrimSpace(amountEntry.Text)
			amount, err := strconv.Atoi(amountText)
			if err != nil || amount < 0 {
				dialog.ShowError(fmt.Errorf("invalid heal amount %q", amountText), w)
				return
			}

			_, _, err = svc.ExecuteHeal(ctx, appsvc.HealCommand{
				CombatantID: target.ID,
				Amount:      amount,
			})
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			if refresh != nil {
				refresh()
			}
		},
		w,
	)
	healDialog.Resize(fyne.NewSize(420, 200))
	healDialog.Show()
}
