package fyneui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func showCampaignEditorDialog(
	w fyne.Window,
	title string,
	submitLabel string,
	initialName string,
	initialStartDate string,
	initialPlayers []domain.NewCampaignPlayer,
	onSubmit func(name string, startDate time.Time, players []domain.NewCampaignPlayer) error,
	refresh func(),
) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g. Commonwealth Survival")
	nameEntry.SetText(strings.TrimSpace(initialName))
	startDateEntry := widget.NewEntry()
	startDateEntry.SetPlaceHolder("YYYY-MM-DD")
	if strings.TrimSpace(initialStartDate) == "" {
		startDateEntry.SetText(time.Now().Format("2006-01-02"))
	} else {
		startDateEntry.SetText(strings.TrimSpace(initialStartDate))
	}

	var rows []*campaignPlayerInputRow
	rowsBox := container.NewVBox()
	headers := container.NewGridWithColumns(
		11,
		newTableHeaderLabel("Player"),
		newTableHeaderLabel("Character"),
		newTableHeaderLabel("Level"),
		newTableHeaderLabel("Init"),
		newTableHeaderLabel("HP Cur"),
		newTableHeaderLabel("HP Max"),
		newTableHeaderLabel("DEF Base"),
		newTableHeaderLabel("DR Poison"),
		newTableHeaderLabel("Details"),
		newTableHeaderLabel("Active"),
		newTableHeaderLabel("Action"),
	)
	table := container.NewVBox(headers, widget.NewSeparator(), rowsBox)
	addRow := func() *campaignPlayerInputRow {
		row := newCampaignPlayerInputRow(func(target *campaignPlayerInputRow) {
			if len(rows) == 1 {
				resetCampaignPlayerInputRow(target)
				return
			}
			filtered := make([]*campaignPlayerInputRow, 0, len(rows)-1)
			for _, r := range rows {
				if r != target {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
			rowsBox.Remove(target.root)
			rowsBox.Refresh()
		})
		rows = append(rows, row)
		rowsBox.Add(row.root)
		rowsBox.Refresh()
		return row
	}
	if len(initialPlayers) == 0 {
		addRow()
	} else {
		for _, p := range initialPlayers {
			row := addRow()
			populateCampaignPlayerInputRow(row, p)
		}
	}

	validationError := widget.NewLabel("")
	validationError.TextStyle = fyne.TextStyle{Monospace: true}
	validationError.Wrapping = fyne.TextWrapWord
	validationError.Importance = widget.DangerImportance
	addPlayerBtn := newRoleButton("+ Add Player", uiActionSecondary, func() { addRow() })
	scroll := container.NewScroll(table)
	dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
	scroll.Direction = container.ScrollBoth
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))

	form := widget.NewForm(
		widget.NewFormItem("Campaign Name", nameEntry),
		widget.NewFormItem("Start Date", startDateEntry),
	)
	playersHeader := container.NewBorder(nil, nil, nil, addPlayerBtn, newDialogSectionLabel("Players"))
	formContent := container.NewVBox(form, widget.NewSeparator(), playersHeader, scroll, widget.NewSeparator(), validationError)

	var editorDialog *dialog.CustomDialog
	cancelBtn := newRoleButton("Cancel", uiActionSubtle, func() { editorDialog.Hide() })
	submitBtn := newRoleButton(submitLabel, uiActionPrimary, func() {
		validationError.SetText("")
		campaignName := strings.TrimSpace(nameEntry.Text)
		startDate := strings.TrimSpace(startDateEntry.Text)
		if campaignName == "" {
			validationError.SetText("Campaign name is required")
			return
		}
		parsedStartDate, err := domain.ParseCampaignStartDate(startDate)
		if err != nil {
			validationError.SetText("Start date must be in YYYY-MM-DD format")
			return
		}

		players, err := collectCampaignPlayersFromRows(rows)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}
		if err := onSubmit(campaignName, parsedStartDate, players); err != nil {
			validationError.SetText(err.Error())
			return
		}
		if refresh != nil {
			refresh()
		}
		editorDialog.Hide()
	})

	editorDialog = dialog.NewCustomWithoutButtons(title, formContent, w)
	editorDialog.SetButtons([]fyne.CanvasObject{cancelBtn, submitBtn})
	editorDialog.Resize(dialogSize)
	editorDialog.Show()
}

func resetCampaignPlayerInputRow(row *campaignPlayerInputRow) {
	row.playerName.SetText("")
	row.characterName.SetText("")
	row.level.SetText("1")
	row.initiative.SetText("1")
	row.hp.SetText("1")
	row.hpMax.SetText("1")
	row.defense.SetText("0")
	row.notes.SetText("")
	for _, attribute := range domain.SpecialAttributes() {
		row.special[attribute].SetText("1")
	}
	row.resistance.reset()
	row.active.SetChecked(true)
}

func populateCampaignPlayerInputRow(row *campaignPlayerInputRow, player domain.NewCampaignPlayer) {
	row.playerName.SetText(player.PlayerName)
	row.characterName.SetText(player.Character.Name)
	row.level.SetText(strconv.Itoa(player.Character.Level))
	row.initiative.SetText(strconv.Itoa(player.Character.Initiative))
	row.hp.SetText(strconv.Itoa(player.Character.HP))
	maxHP := player.Character.MaxHP
	if maxHP <= 0 {
		maxHP = player.Character.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	row.hpMax.SetText(strconv.Itoa(maxHP))
	row.defense.SetText(strconv.Itoa(player.Character.Defense))
	row.notes.SetText(player.Notes)
	special := player.Special
	if special.IsZero() {
		special = domain.DefaultSpecialValues()
	}
	for _, attribute := range domain.SpecialAttributes() {
		row.special[attribute].SetText(strconv.Itoa(special.Value(attribute)))
	}
	row.resistance.setProfile(player.Character.ResistanceProfile())
	row.active.SetChecked(!player.Inactive)
}

func showCreateCampaignDialogWindow(ctx context.Context, w fyne.Window, svc *appsvc.Service, refresh func()) {
	showCampaignEditorDialog(
		w,
		"Create Campaign",
		"Create",
		"",
		time.Now().Format("2006-01-02"),
		nil,
		func(name string, startDate time.Time, players []domain.NewCampaignPlayer) error {
			_, err := svc.CreateCampaign(ctx, "", name, startDate, players)
			return err
		},
		refresh,
	)
}

func showCampaignListDialogWindow(
	ctx context.Context,
	w fyne.Window,
	svc *appsvc.Service,
	activeCampaign *domain.Campaign,
	showCreateCampaignDialog func(),
	refresh func(),
	handleErr func(error),
) {
	campaigns, err := svc.ListCampaigns(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	if len(campaigns) == 0 {
		dialog.ShowInformation("Campaigns", "No campaigns yet. Create one first.", w)
		return
	}

	selectedID := campaigns[0].ID
	selectedInfo := widget.NewLabel("")
	selectedInfo.TextStyle = fyne.TextStyle{Monospace: true}
	selectedInfo.Wrapping = fyne.TextWrapWord
	selectedIdx := 0

	renderSelected := func(idx int) {
		if idx < 0 || idx >= len(campaigns) {
			selectedID = ""
			selectedInfo.SetText("No campaign selected")
			return
		}
		selectedIdx = idx
		selectedID = campaigns[idx].ID
		current := ""
		if activeCampaign != nil && activeCampaign.ID == campaigns[idx].ID {
			current = " (active)"
		}
		selectedInfo.SetText(fmt.Sprintf(
			"Name: %s%s\nID: %s\nStart Date: %s\nUpdated: %s",
			campaigns[idx].Name, current, campaigns[idx].ID, formatCampaignStartDate(campaigns[idx].StartDate), formatTimestamp(campaigns[idx].UpdatedAt),
		))
	}

	list := widget.NewList(
		func() int { return len(campaigns) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("campaign")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Wrapping = fyne.TextWrapOff
			label.Truncation = fyne.TextTruncateClip
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := campaigns[i]
			activeMark := ""
			if activeCampaign != nil && activeCampaign.ID == c.ID {
				activeMark = " [active]"
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s%s | Start:%s | Updated:%s", c.Name, activeMark, formatCampaignStartDate(c.StartDate), formatTimestamp(c.UpdatedAt)))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { renderSelected(id) }

	dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width*0.48, dialogSize.Height*0.62))

	var campaignDialog *dialog.CustomDialog
	createBtn := newRoleButton("Create New", uiActionSecondary, func() {
		campaignDialog.Hide()
		showCreateCampaignDialog()
	})
	editBtn := newRoleButton("Edit", uiActionSubtle, func() {
		if selectedID == "" || selectedIdx < 0 || selectedIdx >= len(campaigns) {
			return
		}
		players, err := svc.ListCampaignPlayers(ctx, selectedID)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		current := campaigns[selectedIdx]
		campaignDialog.Hide()
		showCampaignEditorDialog(
			w,
			"Edit Campaign",
			"Save",
			current.Name,
			formatCampaignStartDate(current.StartDate),
			players,
			func(name string, startDate time.Time, editedPlayers []domain.NewCampaignPlayer) error {
				_, updateErr := svc.UpdateCampaign(ctx, current.ID, name, startDate, editedPlayers)
				return updateErr
			},
			refresh,
		)
	})
	infoBtn := newRoleButton("Use Selected", uiActionPrimary, func() {
		if selectedIdx >= 0 && selectedIdx < len(campaigns) {
			if _, err := svc.ActivateCampaign(ctx, campaigns[selectedIdx].ID); err != nil {
				dialog.ShowError(err, w)
				return
			}
			if refresh != nil {
				refresh()
			}
			campaignDialog.Hide()
		}
	})

	renderSelected(0)
	list.Select(0)

	actions := container.NewGridWithColumns(3, infoBtn, editBtn, createBtn)
	detailPane := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), actions),
		nil,
		nil,
		container.NewPadded(selectedInfo),
	)
	detailPane.Resize(fyne.NewSize(dialogSize.Width*0.42, dialogSize.Height*0.62))
	split := container.NewHSplit(scroll, detailPane)
	split.Offset = 0.55

	content := container.NewBorder(nil, nil, nil, nil, split)

	campaignDialog = dialog.NewCustom("Campaigns", "Close", content, w)
	campaignDialog.Resize(dialogSize)
	campaignDialog.Show()
}
