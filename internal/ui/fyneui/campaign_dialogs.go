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
		newTableHeaderLabel("DR Details"),
		newTableHeaderLabel("Active"),
		newTableHeaderLabel("Action"),
	)
	table := container.NewVBox(headers, widget.NewSeparator(), rowsBox)
	addRow := func() *campaignPlayerInputRow {
		row := newCampaignPlayerInputRow(func(target *campaignPlayerInputRow) {
			if len(rows) == 1 {
				target.playerName.SetText("")
				target.characterName.SetText("")
				target.level.SetText("1")
				target.initiative.SetText("1")
				target.hp.SetText("1")
				target.hpMax.SetText("1")
				target.defense.SetText("0")
				target.drEnergyHead.SetText("0")
				target.drEnergyTorso.SetText("0")
				target.drEnergyLA.SetText("0")
				target.drEnergyRA.SetText("0")
				target.drEnergyLL.SetText("0")
				target.drEnergyRL.SetText("0")
				target.drRadHead.SetText("0")
				target.drRadTorso.SetText("0")
				target.drRadLA.SetText("0")
				target.drRadRA.SetText("0")
				target.drRadLL.SetText("0")
				target.drRadRL.SetText("0")
				target.drPhysHead.SetText("0")
				target.drPhysTorso.SetText("0")
				target.drPhysLA.SetText("0")
				target.drPhysRA.SetText("0")
				target.drPhysLL.SetText("0")
				target.drPhysRL.SetText("0")
				target.immPhysical.SetChecked(false)
				target.immEnergy.SetChecked(false)
				target.immRadiation.SetChecked(false)
				target.drPoison.SetText("0")
				target.immPoison.SetChecked(false)
				target.active.SetChecked(true)
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
			row.playerName.SetText(p.PlayerName)
			row.characterName.SetText(p.Character.Name)
			row.level.SetText(strconv.Itoa(p.Character.Level))
			row.initiative.SetText(strconv.Itoa(p.Character.Initiative))
			row.hp.SetText(strconv.Itoa(p.Character.HP))
			maxHP := p.Character.MaxHP
			if maxHP <= 0 {
				maxHP = p.Character.HP
			}
			if maxHP <= 0 {
				maxHP = 1
			}
			row.hpMax.SetText(strconv.Itoa(maxHP))
			row.defense.SetText(strconv.Itoa(p.Character.Defense))
			row.drEnergyHead.SetText(strconv.Itoa(p.Character.ResistEnergyHead))
			row.drEnergyTorso.SetText(strconv.Itoa(p.Character.ResistEnergyTorso))
			row.drEnergyLA.SetText(strconv.Itoa(p.Character.ResistEnergyLeftArm))
			row.drEnergyRA.SetText(strconv.Itoa(p.Character.ResistEnergyRightArm))
			row.drEnergyLL.SetText(strconv.Itoa(p.Character.ResistEnergyLeftLeg))
			row.drEnergyRL.SetText(strconv.Itoa(p.Character.ResistEnergyRightLeg))
			row.drRadHead.SetText(strconv.Itoa(p.Character.ResistRadiationHead))
			row.drRadTorso.SetText(strconv.Itoa(p.Character.ResistRadiationTorso))
			row.drRadLA.SetText(strconv.Itoa(p.Character.ResistRadiationLeftArm))
			row.drRadRA.SetText(strconv.Itoa(p.Character.ResistRadiationRightArm))
			row.drRadLL.SetText(strconv.Itoa(p.Character.ResistRadiationLeftLeg))
			row.drRadRL.SetText(strconv.Itoa(p.Character.ResistRadiationRightLeg))
			row.immPhysical.SetChecked(p.Character.ImmunePhysical)
			if !p.Character.ImmunePhysical {
				row.drPhysHead.SetText(strconv.Itoa(p.Character.ResistPhysicalHead))
				row.drPhysTorso.SetText(strconv.Itoa(p.Character.ResistPhysicalTorso))
				row.drPhysLA.SetText(strconv.Itoa(p.Character.ResistPhysicalLeftArm))
				row.drPhysRA.SetText(strconv.Itoa(p.Character.ResistPhysicalRightArm))
				row.drPhysLL.SetText(strconv.Itoa(p.Character.ResistPhysicalLeftLeg))
				row.drPhysRL.SetText(strconv.Itoa(p.Character.ResistPhysicalRightLeg))
			}
			row.immEnergy.SetChecked(p.Character.ImmuneEnergy)
			row.immRadiation.SetChecked(p.Character.ImmuneRadiation)
			row.immPoison.SetChecked(p.Character.ImmunePoison)
			if !p.Character.ImmunePoison {
				row.drPoison.SetText(strconv.Itoa(p.Character.ResistPoison))
			}
			row.active.SetChecked(!p.Inactive)
		}
	}

	validationError := widget.NewLabel("")
	validationError.TextStyle = fyne.TextStyle{Monospace: true}
	validationError.Wrapping = fyne.TextWrapWord
	addPlayerBtn := widget.NewButton("+ Add Player", func() { addRow() })
	scroll := container.NewScroll(table)
	dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
	scroll.Direction = container.ScrollBoth
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))
	playerSection := container.NewVBox(addPlayerBtn, scroll)

	form := widget.NewForm(
		widget.NewFormItem("Campaign Name", nameEntry),
		widget.NewFormItem("Start Date", startDateEntry),
		widget.NewFormItem("Players", playerSection),
	)
	formContent := container.NewVBox(form, widget.NewSeparator(), validationError)

	var editorDialog *dialog.CustomDialog
	cancelBtn := widget.NewButton("Cancel", func() { editorDialog.Hide() })
	submitBtn := widget.NewButton(submitLabel, func() {
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
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.45))

	var campaignDialog *dialog.CustomDialog
	activateBtn := widget.NewButton("Activate", func() {
		if selectedID == "" {
			return
		}
		if _, err := svc.ActivateCampaign(ctx, selectedID); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if refresh != nil {
			refresh()
		}
		campaignDialog.Hide()
	})
	createBtn := widget.NewButton("Create New", func() {
		campaignDialog.Hide()
		showCreateCampaignDialog()
	})
	editBtn := widget.NewButton("Edit", func() {
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
	infoBtn := widget.NewButton("Use Selected", func() {
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

	content := container.NewVBox(
		scroll,
		widget.NewSeparator(),
		selectedInfo,
		widget.NewSeparator(),
		container.NewGridWithColumns(4, activateBtn, infoBtn, editBtn, createBtn),
	)

	campaignDialog = dialog.NewCustom("Campaigns", "Close", content, w)
	campaignDialog.Resize(dialogSize)
	campaignDialog.Show()
}
