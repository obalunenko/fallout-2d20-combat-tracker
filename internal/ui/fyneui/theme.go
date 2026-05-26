package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type pipBoyTheme struct{}

var (
	pipColorBackground       = color.NRGBA{R: 3, G: 20, B: 8, A: 255}
	pipColorSurface          = color.NRGBA{R: 5, G: 26, B: 11, A: 236}
	pipColorSurfaceStrong    = color.NRGBA{R: 11, G: 45, B: 20, A: 255}
	pipColorSurfaceHover     = color.NRGBA{R: 20, G: 70, B: 31, A: 255}
	pipColorSurfacePressed   = color.NRGBA{R: 29, G: 100, B: 46, A: 255}
	pipColorInput            = color.NRGBA{R: 8, G: 32, B: 15, A: 255}
	pipColorOverlay          = color.NRGBA{R: 20, G: 24, B: 33, A: 255}
	pipColorForeground       = color.NRGBA{R: 140, G: 255, B: 175, A: 255}
	pipColorForegroundMuted  = color.NRGBA{R: 86, G: 148, B: 105, A: 255}
	pipColorForegroundSubtle = color.NRGBA{R: 90, G: 152, B: 110, A: 255}
	pipColorPrimary          = color.NRGBA{R: 99, G: 255, B: 145, A: 255}
	pipColorPrimaryDim       = color.NRGBA{R: 23, G: 88, B: 41, A: 190}
	pipColorWarning          = color.NRGBA{R: 255, G: 190, B: 95, A: 255}
	pipColorDanger           = color.NRGBA{R: 255, G: 95, B: 95, A: 255}
	pipColorSuccess          = color.NRGBA{R: 116, G: 245, B: 160, A: 255}
	pipColorTextOnAccent     = color.NRGBA{R: 3, G: 20, B: 8, A: 255}
	pipColorSeparator        = color.NRGBA{R: 70, G: 140, B: 93, A: 120}
	pipColorScrollBar        = color.NRGBA{R: 62, G: 170, B: 96, A: 180}
	pipColorScanlineGlow     = color.NRGBA{R: 38, G: 125, B: 66, A: 20}
	pipColorScanlineBright   = color.NRGBA{R: 150, G: 255, B: 180, A: 16}
)

func newPipBoyTheme() fyne.Theme {
	return &pipBoyTheme{}
}

func (t *pipBoyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return pipColorBackground
	case theme.ColorNameForeground:
		return pipColorForeground
	case theme.ColorNamePrimary:
		return pipColorPrimary
	case theme.ColorNameForegroundOnPrimary:
		return pipColorTextOnAccent
	case theme.ColorNameButton:
		return pipColorSurfaceStrong
	case theme.ColorNameHover:
		return pipColorSurfaceHover
	case theme.ColorNamePressed:
		return pipColorSurfacePressed
	case theme.ColorNameFocus:
		return pipColorPrimary
	case theme.ColorNameInputBackground:
		return pipColorInput
	case theme.ColorNameInputBorder:
		return pipColorSeparator
	case theme.ColorNamePlaceHolder:
		return pipColorForegroundSubtle
	case theme.ColorNameSelection:
		return pipColorPrimaryDim
	case theme.ColorNameSeparator:
		return pipColorSeparator
	case theme.ColorNameScrollBar:
		return pipColorScrollBar
	case theme.ColorNameScrollBarBackground:
		return pipColorInput
	case theme.ColorNameDisabled:
		return pipColorForegroundMuted
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 11, G: 31, B: 19, A: 255}
	case theme.ColorNameError:
		return pipColorDanger
	case theme.ColorNameForegroundOnError:
		return pipColorTextOnAccent
	case theme.ColorNameWarning:
		return pipColorWarning
	case theme.ColorNameForegroundOnWarning:
		return pipColorTextOnAccent
	case theme.ColorNameSuccess:
		return pipColorSuccess
	case theme.ColorNameForegroundOnSuccess:
		return pipColorTextOnAccent
	case theme.ColorNameHeaderBackground:
		return pipColorSurface
	case theme.ColorNameOverlayBackground:
		return pipColorOverlay
	case theme.ColorNameHyperlink:
		return pipColorWarning
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 180}
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
