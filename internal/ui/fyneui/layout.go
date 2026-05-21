package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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

	header := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
	)
	content := container.NewBorder(header, nil, nil, nil, body)

	panelBG := canvas.NewRectangle(color.NRGBA{R: 5, G: 26, B: 11, A: 236})
	return container.NewStack(panelBG, container.NewPadded(content))
}

func newScanlineOverlay() fyne.CanvasObject {
	scan := canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		if y%3 == 0 {
			return color.NRGBA{R: 150, G: 255, B: 180, A: 16}
		}
		if y%7 == 0 && x%2 == 0 {
			return color.NRGBA{R: 0, G: 0, B: 0, A: 12}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	})
	return scan
}
