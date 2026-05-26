package fyneui

import "fyne.io/fyne/v2/widget"

type uiActionRole int

const (
	uiActionSecondary uiActionRole = iota
	uiActionPrimary
	uiActionSubtle
	uiActionDestructive
	uiActionWarning
	uiActionSuccess
)

func newRoleButton(label string, role uiActionRole, tapped func()) *widget.Button {
	return styleButtonRole(widget.NewButton(label, tapped), role)
}

func styleButtonRole(button *widget.Button, role uiActionRole) *widget.Button {
	if button == nil {
		return nil
	}
	button.Importance = buttonImportanceForRole(role)
	return button
}

func buttonImportanceForRole(role uiActionRole) widget.Importance {
	switch role {
	case uiActionPrimary:
		return widget.HighImportance
	case uiActionSubtle:
		return widget.LowImportance
	case uiActionDestructive:
		return widget.DangerImportance
	case uiActionWarning:
		return widget.WarningImportance
	case uiActionSuccess:
		return widget.SuccessImportance
	default:
		return widget.MediumImportance
	}
}

func combatantImportance(cDefeated, cActive, cNeedsAttention bool) widget.Importance {
	switch {
	case cDefeated:
		return widget.DangerImportance
	case cActive:
		return widget.HighImportance
	case cNeedsAttention:
		return widget.WarningImportance
	default:
		return widget.LowImportance
	}
}
