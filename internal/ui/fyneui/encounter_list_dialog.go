package fyneui

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func showEncounterListDialogWindow(
	ctx context.Context,
	w fyne.Window,
	svc *appsvc.Service,
	refresh func(),
	handleErr func(error),
	// track registers reload so assistant changes refresh the open list; it
	// returns the function that unregisters it. Nil disables live reload.
	track func(reload func()) (untrack func()),
) {
	summaries, err := svc.ListEncounters(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	if len(summaries) == 0 {
		dialog.ShowInformation("Encounters", "No saved encounters yet.", w)
		return
	}

	selectedID := summaries[0].ID
	selectedInfo := widget.NewLabel("")
	selectedInfo.TextStyle = fyne.TextStyle{Monospace: true}
	selectedInfo.Wrapping = fyne.TextWrapWord
	selectedIdx := 0

	var list *widget.List
	var launchBtn *widget.Button
	var restartBtn *widget.Button
	var deleteBtn *widget.Button
	var editBtn *widget.Button

	renderSelected := func(idx int) {
		if idx < 0 || idx >= len(summaries) {
			selectedID = ""
			selectedInfo.SetText("No encounter selected")
			return
		}
		s := summaries[idx]
		selectedIdx = idx
		selectedID = s.ID
		selectedInfo.SetText(
			fmt.Sprintf(
				"Name: %s\nID: %s\nRound: %d\nCombatants: %d\nDifficulty: %s\nUpdated: %s",
				s.Name, s.ID, s.Round, s.Combatants, formatEncounterDifficultySummary(s), formatTimestamp(s.UpdatedAt),
			),
		)
	}

	updateActionButtons := func() {
		if selectedID == "" {
			launchBtn.Disable()
			restartBtn.Disable()
			deleteBtn.Disable()
			editBtn.Disable()
			return
		}
		launchBtn.Enable()
		restartBtn.Enable()
		deleteBtn.Enable()
		editBtn.Enable()
	}

	list = widget.NewList(
		func() int { return len(summaries) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("encounter")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Wrapping = fyne.TextWrapOff
			label.Truncation = fyne.TextTruncateClip
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			s := summaries[i]
			o.(*widget.Label).SetText(
				fmt.Sprintf(
					"%s | %s | Round:%d | Combatants:%d | Updated:%s",
					s.Name, formatEncounterDifficultySummary(s), s.Round, s.Combatants, formatTimestamp(s.UpdatedAt),
				),
			)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		renderSelected(id)
		updateActionButtons()
	}

	dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width*0.5, dialogSize.Height*0.62))

	// showSummaries shows updated and selects keepID, or the first row when
	// keepID is empty or not listed.
	showSummaries := func(updated []domain.EncounterSummary, keepID string) {
		summaries = updated
		list.Refresh()

		if len(summaries) == 0 {
			selectedID = ""
			selectedInfo.SetText("No saved encounters left")
			updateActionButtons()
			return
		}

		nextIdx := 0
		if keepID != "" {
			for i := range summaries {
				if summaries[i].ID == keepID {
					nextIdx = i
					break
				}
			}
		}
		renderSelected(nextIdx)
		list.Select(nextIdx)
		updateActionButtons()
	}

	refreshSummaries := func(keepID string) error {
		updated, err := svc.ListEncounters(ctx)
		if err != nil {
			return err
		}
		showSummaries(updated, keepID)
		return nil
	}

	// reload re-reads the list after an assistant change. It keeps the
	// selection, and clears it with a note when the encounter was deleted.
	reload := func() {
		prevID := selectedID
		updated, err := svc.ListEncounters(ctx)
		if err != nil {
			return
		}
		if prevID == "" || slices.ContainsFunc(updated, func(s domain.EncounterSummary) bool { return s.ID == prevID }) {
			showSummaries(updated, prevID)
			return
		}
		summaries = updated
		list.UnselectAll()
		list.Refresh()
		selectedID = ""
		selectedInfo.SetText("The selected encounter was deleted by the assistant.")
		updateActionButtons()
	}

	var encounterDialog *dialog.CustomDialog

	launchBtn = newRoleButton("Launch", uiActionPrimary, func() {
		if selectedID == "" {
			return
		}
		_, err := svc.ActivateEncounter(ctx, selectedID)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		if refresh != nil {
			refresh()
		}
		encounterDialog.Hide()
	})
	restartBtn = newRoleButton("Restart", uiActionWarning, func() {
		if selectedID == "" {
			return
		}
		_, err := svc.RestartEncounter(ctx, selectedID)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		if refresh != nil {
			refresh()
		}
		encounterDialog.Hide()
	})
	deleteBtn = newRoleButton("Delete", uiActionDestructive, func() {
		if selectedID == "" {
			return
		}
		targetID := selectedID
		targetName := summaries[selectedIdx].Name
		dialog.ShowConfirm(
			"Delete Encounter",
			fmt.Sprintf("Soft delete encounter %q?", targetName),
			func(ok bool) {
				if !ok {
					return
				}

				if err := svc.DeleteEncounter(ctx, targetID); err != nil {
					dialog.ShowError(err, w)
					return
				}

				if err := refreshSummaries(""); err != nil {
					dialog.ShowError(err, w)
					return
				}
				if refresh != nil {
					refresh()
				}
				if len(summaries) == 0 {
					encounterDialog.Hide()
				}
			},
			w,
		)
	})
	editBtn = newRoleButton("Edit", uiActionSecondary, func() {
		if selectedID == "" {
			return
		}
		targetID := selectedID
		encounterDialog.Hide()
		openEncounterEditor(ctx, w, svc, targetID, refresh)
	})

	renderSelected(0)
	list.Select(0)
	updateActionButtons()

	primaryActions := container.NewGridWithColumns(2, launchBtn, editBtn)
	secondaryActions := container.NewGridWithColumns(2, restartBtn, deleteBtn)
	detailPane := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), primaryActions, secondaryActions),
		nil,
		nil,
		container.NewPadded(selectedInfo),
	)
	detailPane.Resize(fyne.NewSize(dialogSize.Width*0.4, dialogSize.Height*0.62))
	split := container.NewHSplit(scroll, detailPane)
	split.Offset = 0.56

	content := container.NewBorder(nil, nil, nil, nil, split)

	encounterDialog = dialog.NewCustom("Encounters", "Close", content, w)
	if track != nil {
		untrack := track(reload)
		encounterDialog.SetOnClosed(untrack)
	}
	encounterDialog.Resize(dialogSize)
	encounterDialog.Show()
}

// openEncounterEditor opens the edit dialog for encounter id. It captures the
// encounter revision before reading, so a save over an assistant change made
// since offers Reload or Overwrite, and a save after an assistant delete only
// closes (FR-022).
func openEncounterEditor(ctx context.Context, w fyne.Window, svc *appsvc.Service, id string, refresh func()) {
	key := appsvc.EntityKey{Kind: domain.EntityEncounter, ID: id}
	opened := svc.Revision(key)
	enc, err := svc.GetEncounterByID(ctx, id)
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	showEncounterEditorDialog(
		ctx,
		w,
		svc,
		fmt.Sprintf("Edit Encounter: %s", enc.Name),
		"Save",
		enc.Name,
		enc.Combatants,
		func(name string, combatants []domain.Combatant) error {
			save := func(expected uint64) error {
				_, err := svc.ExecuteUpdateEncounter(ctx, appsvc.UpdateEncounterCommand{
					EncounterID: id, Name: name, Combatants: combatants, ExpectedRevision: expected,
				})
				return err
			}
			err := save(opened)
			if errors.Is(err, domain.ErrStaleEdit) {
				return &staleEdit{
					entity:    "encounter",
					overwrite: func() error { return save(svc.Revision(key)) },
					reload:    func() { openEncounterEditor(ctx, w, svc, id, refresh) },
				}
			}
			return err
		},
		refresh,
	)
}
