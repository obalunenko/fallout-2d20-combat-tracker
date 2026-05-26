package fyneui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

func dynamicEncounterDialogSize(canvasSize fyne.Size) fyne.Size {
	width := canvasSize.Width * 0.94
	height := canvasSize.Height * 0.84

	if width < 860 {
		width = 860
	}
	if height < 480 {
		height = 480
	}
	return fyne.NewSize(width, height)
}

func pipPanel(title string, body fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabel("> " + title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	titleLabel.Importance = widget.HighImportance

	header := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
	)
	content := container.NewBorder(header, nil, nil, nil, body)

	panelBG := canvas.NewRectangle(pipColorSurface)
	return container.NewStack(panelBG, container.NewPadded(content))
}

func newMonospaceLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	return label
}

func newWrappedMonospaceLabel(text string) *widget.Label {
	label := newMonospaceLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func newDialogSectionLabel(text string) *widget.Label {
	label := newMonospaceLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	label.Importance = widget.HighImportance
	return label
}

func newReadOnlyMonospaceOutput(initialText string, minRows int) *widget.Entry {
	output := widget.NewMultiLineEntry()
	output.TextStyle = fyne.TextStyle{Monospace: true}
	output.Wrapping = fyne.TextWrapWord
	output.SetMinRowsVisible(minRows)
	output.Disable()
	output.SetText(initialText)
	return output
}

func newMainContentWithHeader(
	campaignStatusLabel *widget.Label,
	mainView fyne.CanvasObject,
	openCampaignBtn,
	openEncounterBtn,
	newCampaignBtn,
	newEncounterBtn *widget.Button,
) fyne.CanvasObject {
	header := widget.NewLabel("PIP-BOY // FALLOUT 2D20 COMBAT TRACKER")
	header.Alignment = fyne.TextAlignCenter
	header.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	header.Importance = widget.HighImportance
	leftControls := container.NewHBox(openCampaignBtn, openEncounterBtn)
	rightControls := container.NewHBox(newCampaignBtn, newEncounterBtn)
	headerBar := container.NewBorder(nil, nil, leftControls, rightControls, header)
	topBar := container.NewVBox(headerBar, campaignStatusLabel)
	return container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
		nil,
		nil,
		nil,
		mainView,
	)
}

func newPipBackground(content fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(pipColorBackground)
	glow := canvas.NewRectangle(pipColorScanlineGlow)
	return container.NewStack(background, content, glow, newScanlineOverlay())
}

func refreshResourceLabels(enc *domain.Encounter, partyAPLabel, threatLabel *widget.Label) {
	if enc == nil {
		partyAPLabel.SetText("Party AP: 0")
		threatLabel.SetText("GM Threat: 0")
		return
	}
	partyAPLabel.SetText(fmt.Sprintf("Party AP: %d", enc.Resources.PartyAP))
	threatLabel.SetText(fmt.Sprintf("GM Threat: %d", enc.Resources.GMThreat))
}

func newScanlineOverlay() fyne.CanvasObject {
	scan := canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		if y%3 == 0 {
			return pipColorScanlineBright
		}
		if y%7 == 0 && x%2 == 0 {
			return color.NRGBA{R: 0, G: 0, B: 0, A: 12}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	})
	return scan
}
