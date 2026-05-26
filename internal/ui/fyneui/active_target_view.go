package fyneui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type activeTargetView struct {
	root      fyne.CanvasObject
	accordion *widget.Accordion

	detailsLabel *widget.Label
	damageBtn    *widget.Button
	healBtn      *widget.Button

	nameLabel    *widget.Label
	sideLabel    *widget.Label
	levelLabel   *widget.Label
	xpLabel      *widget.Label
	statusLabel  *widget.Label
	hpLabel      *widget.Label
	initLabel    *widget.Label
	defenseLabel *widget.Label
	poisonLabel  *widget.Label
	physLabel    *widget.Label
	energyLabel  *widget.Label
	radLabel     *widget.Label
	hpBar        *widget.ProgressBar
}

func newActiveTargetView(detailsLabel *widget.Label, damageBtn, healBtn *widget.Button) *activeTargetView {
	if detailsLabel == nil {
		detailsLabel = newWrappedMonospaceLabel("")
	}
	detailsLabel.TextStyle = fyne.TextStyle{Monospace: true}

	view := &activeTargetView{
		detailsLabel: detailsLabel,
		damageBtn:    damageBtn,
		healBtn:      healBtn,
		nameLabel:    newActiveTargetValueLabel(widget.HighImportance),
		sideLabel:    newActiveTargetValueLabel(widget.MediumImportance),
		levelLabel:   newActiveTargetValueLabel(widget.MediumImportance),
		xpLabel:      newActiveTargetValueLabel(widget.MediumImportance),
		statusLabel:  newActiveTargetValueLabel(widget.MediumImportance),
		hpLabel:      newActiveTargetValueLabel(widget.MediumImportance),
		initLabel:    newActiveTargetValueLabel(widget.MediumImportance),
		defenseLabel: newActiveTargetValueLabel(widget.MediumImportance),
		poisonLabel:  newActiveTargetValueLabel(widget.LowImportance),
		physLabel:    newActiveTargetValueLabel(widget.MediumImportance),
		energyLabel:  newActiveTargetValueLabel(widget.MediumImportance),
		radLabel:     newActiveTargetValueLabel(widget.MediumImportance),
		hpBar:        widget.NewProgressBar(),
	}
	view.nameLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	view.nameLabel.Wrapping = fyne.TextWrapWord
	view.nameLabel.Truncation = fyne.TextTruncateOff
	view.statusLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	view.statusLabel.Alignment = fyne.TextAlignTrailing

	identity := container.NewGridWithColumns(
		3,
		newActiveTargetMetric("Side", view.sideLabel),
		newActiveTargetMetric("Level", view.levelLabel),
		newActiveTargetMetric("XP", view.xpLabel),
	)
	combat := container.NewGridWithColumns(
		3,
		newActiveTargetMetricWithBody("HP", nil, view.hpBar),
		newActiveTargetMetric("Initiative", view.initLabel),
		newActiveTargetMetric("Defense", view.defenseLabel),
	)
	resistances := container.NewGridWithColumns(
		4,
		newActiveTargetMetric("DR Poison", view.poisonLabel),
		newActiveTargetMetric("DR Physical", view.physLabel),
		newActiveTargetMetric("DR Energy", view.energyLabel),
		newActiveTargetMetric("DR Radiation", view.radLabel),
	)
	actions := container.NewGridWithColumns(2, damageBtn, healBtn)
	view.accordion = widget.NewAccordion(
		widget.NewAccordionItem("TARGET DETAILS", container.NewVBox(view.detailsLabel)),
	)
	header := container.NewBorder(nil, nil, nil, view.statusLabel, view.nameLabel)
	view.root = container.NewVBox(
		header,
		widget.NewSeparator(),
		newActiveTargetSectionLabel("SIGNAL"),
		identity,
		widget.NewSeparator(),
		newActiveTargetSectionLabel("VITALS"),
		combat,
		actions,
		widget.NewSeparator(),
		newActiveTargetSectionLabel("RESISTANCES"),
		resistances,
		widget.NewSeparator(),
		view.accordion,
	)
	view.SetTarget(nil, 0)
	return view
}

func (v *activeTargetView) Root() fyne.CanvasObject {
	return v.root
}

func (v *activeTargetView) SetTarget(enc *domain.Encounter, idx int) {
	if enc == nil || len(enc.Combatants) == 0 {
		v.setEmpty()
		return
	}
	if idx < 0 || idx >= len(enc.Combatants) {
		idx = 0
	}
	c := enc.Combatants[idx]
	v.detailsLabel.SetText(formatActiveTargetDetails(enc, c))

	v.nameLabel.SetText(encounterDisplayNameByID(enc, c.ID))
	v.nameLabel.Importance = combatantImportance(isCombatantDefeated(c), c.Active, combatantNeedsAttention(c))
	v.sideLabel.SetText(strings.ToUpper(string(c.Side)))
	v.sideLabel.Importance = sideImportance(c)
	v.levelLabel.SetText(strconv.Itoa(c.Level))
	v.xpLabel.SetText(strconv.Itoa(c.XP))
	v.statusLabel.SetText(formatCombatantStatus(c))
	v.statusLabel.Importance = combatantImportance(isCombatantDefeated(c), c.Active, combatantNeedsAttention(c))
	hpText := formatCombatantHP(c)
	v.hpLabel.SetText(hpText)
	v.hpLabel.Importance = hpImportance(c)
	v.hpBar.TextFormatter = func() string { return hpText }
	v.initLabel.SetText(strconv.Itoa(c.Initiative))
	v.defenseLabel.SetText(strconv.Itoa(c.Defense))
	v.poisonLabel.SetText(formatCombatantGlobalResistance(c, domain.DamagePoison))
	v.poisonLabel.Importance = poisonImportance(c)
	v.physLabel.SetText(formatCombatantLocationResistance(c, domain.DamagePhysical, domain.BodyTorso))
	v.energyLabel.SetText(formatCombatantLocationResistance(c, domain.DamageEnergy, domain.BodyTorso))
	v.radLabel.SetText(formatCombatantLocationResistance(c, domain.DamageRadiation, domain.BodyTorso))
	v.setHPBar(c)
	v.setActionState(isCombatantDefeated(c), true)
	v.refresh()
}

func (v *activeTargetView) setEmpty() {
	v.detailsLabel.SetText("No combatants")
	v.nameLabel.SetText("No active target")
	v.nameLabel.Importance = widget.LowImportance
	for _, label := range []*widget.Label{
		v.sideLabel,
		v.levelLabel,
		v.xpLabel,
		v.statusLabel,
		v.hpLabel,
		v.initLabel,
		v.defenseLabel,
		v.poisonLabel,
		v.physLabel,
		v.energyLabel,
		v.radLabel,
	} {
		label.SetText("-")
		label.Importance = widget.LowImportance
	}
	v.hpBar.Min = 0
	v.hpBar.Max = 1
	v.hpBar.Value = 0
	v.hpBar.TextFormatter = func() string { return "-" }
	v.setActionState(false, false)
	v.refresh()
}

func (v *activeTargetView) setHPBar(c domain.Combatant) {
	maxHP := c.MaxHP
	if maxHP <= 0 {
		maxHP = c.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	v.hpBar.Min = 0
	v.hpBar.Max = float64(maxHP)
	v.hpBar.Value = float64(c.HP)
	if v.hpBar.Value < v.hpBar.Min {
		v.hpBar.Value = v.hpBar.Min
	}
	if v.hpBar.Value > v.hpBar.Max {
		v.hpBar.Value = v.hpBar.Max
	}
}

func (v *activeTargetView) setActionState(defeated, hasTarget bool) {
	if v.damageBtn != nil {
		if !hasTarget || defeated {
			v.damageBtn.Disable()
		} else {
			v.damageBtn.Enable()
		}
	}
	if v.healBtn != nil {
		if hasTarget {
			v.healBtn.Enable()
		} else {
			v.healBtn.Disable()
		}
	}
}

func (v *activeTargetView) refresh() {
	for _, object := range []fyne.CanvasObject{v.root, v.accordion, v.detailsLabel, v.hpBar} {
		if object != nil {
			object.Refresh()
		}
	}
}

func newActiveTargetMetric(title string, value *widget.Label) fyne.CanvasObject {
	return newActiveTargetMetricWithBody(title, value, nil)
}

func newActiveTargetSectionLabel(text string) *widget.Label {
	label := widget.NewLabel("> " + text)
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	label.Importance = widget.HighImportance
	return label
}

func newActiveTargetMetricWithBody(title string, value *widget.Label, body fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabel(strings.ToUpper(title))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	titleLabel.Importance = widget.LowImportance
	if body == nil {
		return container.NewVBox(titleLabel, value)
	}
	if value == nil {
		return container.NewVBox(titleLabel, body)
	}
	return container.NewVBox(titleLabel, value, body)
}

func newActiveTargetValueLabel(importance widget.Importance) *widget.Label {
	label := widget.NewLabel("-")
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Importance = importance
	label.Wrapping = fyne.TextWrapOff
	label.Truncation = fyne.TextTruncateClip
	return label
}
