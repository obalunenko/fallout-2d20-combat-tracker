package fyneui

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type encounterOrderView struct {
	enc                 **domain.Encounter
	selectedIndex       *int
	expandedCombatantID *string
	selectedLabel       *widget.Label
	list                *widget.List
	orderBox            *fyne.Container
	showApplyDamage     func(int)
	showHeal            func(int)
	onSelect            func(int, bool)
}

func newEncounterOrderView(
	enc **domain.Encounter,
	selectedIndex *int,
	expandedCombatantID *string,
	selectedLabel *widget.Label,
	showApplyDamage func(int),
	showHeal func(int),
) *encounterOrderView {
	v := &encounterOrderView{
		enc:                 enc,
		selectedIndex:       selectedIndex,
		expandedCombatantID: expandedCombatantID,
		selectedLabel:       selectedLabel,
		orderBox:            container.NewVBox(),
		showApplyDamage:     showApplyDamage,
		showHeal:            showHeal,
	}
	v.list = widget.NewList(
		func() int {
			current := v.currentEncounter()
			if current == nil {
				return 0
			}
			return len(current.Combatants)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			current := v.currentEncounter()
			if current == nil || i >= len(current.Combatants) {
				return
			}
			label := o.(*widget.Label)
			c := current.Combatants[i]
			isDefeated := isCombatantDefeated(c)
			label.SetText(formatCombatantLine(current, c))
			label.Importance = combatantImportance(isDefeated, c.Active, combatantNeedsAttention(c))
			if isDefeated {
				label.TextStyle = fyne.TextStyle{Monospace: true, Italic: true}
				return
			}
			label.TextStyle = fyne.TextStyle{Monospace: true, Bold: c.Active}
		},
	)
	v.list.OnSelected = func(id widget.ListItemID) {
		v.selectCombatant(id)
	}
	return v
}

func (v *encounterOrderView) List() *widget.List {
	return v.list
}

func (v *encounterOrderView) OrderBox() fyne.CanvasObject {
	return v.orderBox
}

func (v *encounterOrderView) Rebuild() {
	v.orderBox.Objects = nil
	current := v.currentEncounter()
	if current == nil || len(current.Combatants) == 0 {
		empty := widget.NewLabel("No combatants")
		empty.TextStyle = fyne.TextStyle{Monospace: true}
		v.orderBox.Add(empty)
		v.orderBox.Refresh()
		return
	}

	v.orderBox.Add(newEncounterOrderHeader())
	for i, c := range current.Combatants {
		v.orderBox.Add(v.newCombatantOrderRow(current, i, c))
	}
	v.orderBox.Refresh()
}

func (v *encounterOrderView) SetOnSelect(onSelect func(int, bool)) {
	v.onSelect = onSelect
}

func (v *encounterOrderView) CollapseDetails() {
	if v.expandedCombatantID == nil || *v.expandedCombatantID == "" {
		return
	}
	*v.expandedCombatantID = ""
	v.Rebuild()
}

func (v *encounterOrderView) currentEncounter() *domain.Encounter {
	if v.enc == nil {
		return nil
	}
	return *v.enc
}

func (v *encounterOrderView) selectCombatant(idx int) {
	enc := v.currentEncounter()
	if enc == nil || idx < 0 || idx >= len(enc.Combatants) {
		return
	}
	repeatedSelection := idx == *v.selectedIndex
	*v.selectedIndex = idx
	if v.expandedCombatantID != nil {
		*v.expandedCombatantID = ""
	}
	refreshSelected(v.selectedLabel, enc, *v.selectedIndex)
	if v.onSelect != nil {
		v.onSelect(idx, repeatedSelection)
	}
	v.Rebuild()
}

func (v *encounterOrderView) newCombatantOrderRow(enc *domain.Encounter, idx int, c domain.Combatant) fyne.CanvasObject {
	isDefeated := isCombatantDefeated(c)
	isSelected := idx == *v.selectedIndex
	importance := combatantImportance(isDefeated, c.Active, combatantNeedsAttention(c))

	selectRow := func() {
		v.selectCombatant(idx)
	}

	marker := newEncounterOrderCell(formatCombatantTurnMarker(c, isSelected), importance)
	marker.Alignment = fyne.TextAlignCenter

	nameBtn := newRoleButton(encounterDisplayNameByID(enc, c.ID), uiActionSubtle, selectRow)
	nameBtn.Alignment = widget.ButtonAlignLeading

	damageBtn := newRoleButton("DMG", uiActionDestructive, func() {
		v.CollapseDetails()
		v.showApplyDamage(idx)
	})
	healBtn := newRoleButton("HEAL", uiActionSuccess, func() {
		v.CollapseDetails()
		v.showHeal(idx)
	})
	if isDefeated {
		damageBtn.Disable()
	}

	meta := container.New(
		encounterOrderTableLayout{},
		marker,
		nameBtn,
		newEncounterOrderCell(strings.ToUpper(string(c.Side)), sideImportance(c)),
		newEncounterOrderCell(strconv.Itoa(c.Initiative), importance),
		newEncounterOrderHPCell(c),
		newEncounterOrderCell(strconv.Itoa(c.Defense), widget.MediumImportance),
		newEncounterOrderCell(formatCombatantGlobalResistance(c, domain.DamagePoison), poisonImportance(c)),
		newEncounterOrderCell(formatCombatantStatus(c), importance),
		damageBtn,
		healBtn,
	)
	rowHeader := container.NewStack(
		canvas.NewRectangle(encounterOrderRowColor(c, isSelected)),
		container.NewPadded(meta),
	)
	return container.NewVBox(rowHeader)
}

func newEncounterOrderHeader() fyne.CanvasObject {
	marker := newEncounterOrderHeaderLabel("Turn")
	marker.Alignment = fyne.TextAlignCenter
	content := container.New(
		encounterOrderTableLayout{},
		marker,
		newEncounterOrderHeaderLabel("Name"),
		newEncounterOrderHeaderLabel("Side"),
		newEncounterOrderHeaderLabel("Init"),
		newEncounterOrderHeaderLabel("HP"),
		newEncounterOrderHeaderLabel("DEF"),
		newEncounterOrderHeaderLabel("Poison"),
		newEncounterOrderHeaderLabel("Status"),
		newEncounterOrderHeaderLabel("DMG"),
		newEncounterOrderHeaderLabel("HEAL"),
	)
	return container.NewStack(canvas.NewRectangle(color.NRGBA{R: 99, G: 255, B: 145, A: 18}), container.NewPadded(content))
}

func newEncounterOrderHeaderLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	label.Importance = widget.HighImportance
	return label
}

func newEncounterOrderCell(text string, importance widget.Importance) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Importance = importance
	label.Wrapping = fyne.TextWrapOff
	label.Truncation = fyne.TextTruncateClip
	return label
}

func newEncounterOrderHPCell(c domain.Combatant) fyne.CanvasObject {
	bar := widget.NewProgressBar()
	bar.TextFormatter = func() string { return formatCombatantHP(c) }
	maxHP := c.MaxHP
	if maxHP <= 0 {
		maxHP = c.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	bar.Min = 0
	bar.Max = float64(maxHP)
	bar.Value = float64(c.HP)
	if bar.Value < bar.Min {
		bar.Value = bar.Min
	}
	if bar.Value > bar.Max {
		bar.Value = bar.Max
	}
	return bar
}

type encounterOrderTableLayout struct{}

var (
	encounterOrderColumnMinWidths = []float32{42, 160, 46, 42, 92, 42, 58, 76, 56, 56}
	encounterOrderColumnWeights   = []float32{0, 5, 1, 0, 2, 0, 1, 2, 0, 0}
)

func (encounterOrderTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	widths := encounterOrderColumnWidths(size.Width)
	x := float32(0)
	for i, object := range objects {
		if i >= len(widths) {
			object.Hide()
			continue
		}
		object.Show()
		object.Move(fyne.NewPos(x, 0))
		object.Resize(fyne.NewSize(widths[i], size.Height))
		x += widths[i]
	}
}

func (encounterOrderTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(0)
	for _, object := range objects {
		height = maxFloat32(height, object.MinSize().Height)
	}
	return fyne.NewSize(sumFloat32(encounterOrderColumnMinWidths), height)
}

func encounterOrderColumnWidths(width float32) []float32 {
	widths := append([]float32(nil), encounterOrderColumnMinWidths...)
	minTotal := sumFloat32(widths)
	if width <= 0 {
		return widths
	}
	if width < minTotal {
		scale := width / minTotal
		for i := range widths {
			widths[i] *= scale
		}
		return widths
	}

	extra := width - minTotal
	weightTotal := sumFloat32(encounterOrderColumnWeights)
	if weightTotal <= 0 {
		return widths
	}
	for i, weight := range encounterOrderColumnWeights {
		widths[i] += extra * weight / weightTotal
	}
	return widths
}

func sumFloat32(values []float32) float32 {
	total := float32(0)
	for _, value := range values {
		total += value
	}
	return total
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func formatCombatantHP(c domain.Combatant) string {
	maxHP := c.MaxHP
	if maxHP <= 0 {
		maxHP = c.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	return strconv.Itoa(c.HP) + "/" + strconv.Itoa(maxHP)
}

func formatCombatantTurnMarker(c domain.Combatant, selected bool) string {
	if isCombatantDefeated(c) {
		return "xx"
	}
	if c.Active {
		return ">>"
	}
	if selected {
		return ">"
	}
	return ""
}

func encounterOrderRowColor(c domain.Combatant, selected bool) color.Color {
	switch {
	case isCombatantDefeated(c):
		return color.NRGBA{R: 255, G: 95, B: 95, A: 18}
	case selected:
		return color.NRGBA{R: 99, G: 255, B: 145, A: 44}
	case c.Active:
		return color.NRGBA{R: 99, G: 255, B: 145, A: 30}
	case combatantNeedsAttention(c):
		return color.NRGBA{R: 255, G: 190, B: 95, A: 18}
	default:
		return color.NRGBA{R: 5, G: 26, B: 11, A: 0}
	}
}

func hpImportance(c domain.Combatant) widget.Importance {
	if isCombatantDefeated(c) {
		return widget.DangerImportance
	}
	if combatantNeedsAttention(c) {
		return widget.WarningImportance
	}
	return widget.MediumImportance
}

func poisonImportance(c domain.Combatant) widget.Importance {
	if c.ImmunePoison {
		return widget.SuccessImportance
	}
	if c.ResistPoison > 0 {
		return widget.WarningImportance
	}
	return widget.LowImportance
}

func sideImportance(c domain.Combatant) widget.Importance {
	if c.Side == domain.SideParty {
		return widget.HighImportance
	}
	return widget.MediumImportance
}
