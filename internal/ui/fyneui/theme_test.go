package fyneui

import (
	"testing"

	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
)

func TestPipBoyThemeExposesActionAndStateColors(t *testing.T) {
	pipTheme := newPipBoyTheme()

	assert.Equal(t, pipColorPrimary, pipTheme.Color(theme.ColorNamePrimary, theme.VariantDark))
	assert.Equal(t, pipColorWarning, pipTheme.Color(theme.ColorNameWarning, theme.VariantDark))
	assert.Equal(t, pipColorDanger, pipTheme.Color(theme.ColorNameError, theme.VariantDark))
	assert.Equal(t, pipColorSuccess, pipTheme.Color(theme.ColorNameSuccess, theme.VariantDark))
	assert.Equal(t, pipColorTextOnAccent, pipTheme.Color(theme.ColorNameForegroundOnPrimary, theme.VariantDark))
	assert.Equal(t, pipColorTextOnAccent, pipTheme.Color(theme.ColorNameForegroundOnError, theme.VariantDark))
	assert.Equal(t, pipColorTextOnAccent, pipTheme.Color(theme.ColorNameForegroundOnWarning, theme.VariantDark))
	assert.Equal(t, pipColorTextOnAccent, pipTheme.Color(theme.ColorNameForegroundOnSuccess, theme.VariantDark))
}

func TestPipBoyThemeKeepsSecondaryTextReadable(t *testing.T) {
	pipTheme := newPipBoyTheme()

	assert.Equal(t, pipColorForegroundMuted, pipTheme.Color(theme.ColorNameDisabled, theme.VariantDark))
	assert.Equal(t, pipColorForegroundSubtle, pipTheme.Color(theme.ColorNamePlaceHolder, theme.VariantDark))
}
