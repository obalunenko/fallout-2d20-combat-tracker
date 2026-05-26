package fyneui

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestButtonRolesMapToFyneImportance(t *testing.T) {
	tests := []struct {
		name string
		role uiActionRole
		want widget.Importance
	}{
		{name: "primary", role: uiActionPrimary, want: widget.HighImportance},
		{name: "secondary", role: uiActionSecondary, want: widget.MediumImportance},
		{name: "subtle", role: uiActionSubtle, want: widget.LowImportance},
		{name: "destructive", role: uiActionDestructive, want: widget.DangerImportance},
		{name: "warning", role: uiActionWarning, want: widget.WarningImportance},
		{name: "success", role: uiActionSuccess, want: widget.SuccessImportance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			btn := newRoleButton("Action", tt.role, func() {})
			assert.Equal(t, tt.want, btn.Importance)
		})
	}
}

func TestCombatantImportancePrioritizesCombatStates(t *testing.T) {
	assert.Equal(t, widget.DangerImportance, combatantImportance(true, true, true))
	assert.Equal(t, widget.HighImportance, combatantImportance(false, true, true))
	assert.Equal(t, widget.WarningImportance, combatantImportance(false, false, true))
	assert.Equal(t, widget.LowImportance, combatantImportance(false, false, false))
}
