package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type pipBoyTheme struct{}

func newPipBoyTheme() fyne.Theme {
	return &pipBoyTheme{}
}

func (t *pipBoyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 3, G: 20, B: 8, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 140, G: 255, B: 175, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 99, G: 255, B: 145, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 11, G: 45, B: 20, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 20, G: 70, B: 31, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 29, G: 100, B: 46, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 8, G: 32, B: 15, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 90, G: 152, B: 110, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 23, G: 88, B: 41, A: 190}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 70, G: 140, B: 93, A: 120}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 62, G: 170, B: 96, A: 180}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 66, G: 108, B: 80, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 11, G: 31, B: 19, A: 255}
	case theme.ColorNameError:
		return color.NRGBA{R: 255, G: 95, B: 95, A: 255}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 255, G: 190, B: 95, A: 255}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 140, G: 255, B: 175, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *pipBoyTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTextMonospaceFont()
}

func (t *pipBoyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *pipBoyTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 15
	case theme.SizeNameInlineIcon:
		return 18
	case theme.SizeNameScrollBar:
		return 10
	default:
		return theme.DefaultTheme().Size(name)
	}
}
