package fyneui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type uiGlobalResistanceInput struct {
	entry  *widget.Entry
	immune *widget.Check
	cell   fyne.CanvasObject
}

type uiResistanceInputs struct {
	global     map[domain.DamageType]uiGlobalResistanceInput
	byLocation map[domain.DamageType]map[domain.BodyLocation]*widget.Entry
}

func newUIResistanceInputs(onChanged func()) uiResistanceInputs {
	inputs := uiResistanceInputs{
		global:     make(map[domain.DamageType]uiGlobalResistanceInput),
		byLocation: make(map[domain.DamageType]map[domain.BodyLocation]*widget.Entry),
	}
	for _, damageType := range domain.LocationDamageTypes() {
		inputs.byLocation[damageType] = make(map[domain.BodyLocation]*widget.Entry)
		for _, location := range domain.BodyLocations() {
			inputs.byLocation[damageType][location] = newResistanceEntry(resistancePlaceholder(damageType, location), onChanged)
		}
		inputs.global[damageType] = uiGlobalResistanceInput{
			immune: newGlobalImmunityCheck("immune all", inputs.locationEntries(damageType), onChanged),
		}
	}

	poison := newResistanceEntry("DR Poison", onChanged)
	poisonCell, poisonImmune := newResistanceInputCell(poison, onChanged)
	inputs.global[domain.DamagePoison] = uiGlobalResistanceInput{
		entry:  poison,
		immune: poisonImmune,
		cell:   poisonCell,
	}

	return inputs
}

func newResistanceEntry(placeholder string, onChanged func()) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(placeholder)
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	entry.SetText("0")
	if onChanged != nil {
		entry.OnChanged = func(string) { onChanged() }
	}
	return entry
}

func (inputs uiResistanceInputs) setProfile(profile domain.ResistanceProfile) {
	for _, damageType := range domain.LocationDamageTypes() {
		for _, location := range domain.BodyLocations() {
			value, _ := profile.LocationResistance(damageType, location)
			if entry := inputs.locationEntry(damageType, location); entry != nil {
				entry.SetText(strconv.Itoa(value))
			}
		}
		_, immune, _ := profile.GlobalResistance(damageType)
		if check := inputs.globalImmune(damageType); check != nil {
			check.SetChecked(immune)
		}
	}

	poisonValue, poisonImmune, _ := profile.GlobalResistance(domain.DamagePoison)
	if entry := inputs.globalEntry(domain.DamagePoison); entry != nil {
		entry.SetText(strconv.Itoa(poisonValue))
	}
	if check := inputs.globalImmune(domain.DamagePoison); check != nil {
		check.SetChecked(poisonImmune)
	}
}

func (inputs uiResistanceInputs) reset() {
	for _, entry := range inputs.entries() {
		entry.SetText("0")
	}
	for _, damageType := range domain.DamageTypes() {
		if check := inputs.globalImmune(damageType); check != nil {
			check.SetChecked(false)
		}
	}
}

func (inputs uiResistanceInputs) isZero() bool {
	for _, entry := range inputs.entries() {
		if strings.TrimSpace(entry.Text) != "0" {
			return false
		}
	}
	for _, damageType := range domain.DamageTypes() {
		if check := inputs.globalImmune(damageType); check != nil && check.Checked {
			return false
		}
	}
	return true
}

func (inputs uiResistanceInputs) collectProfile(label string, torsoOnly bool) (domain.ResistanceProfile, error) {
	profile := domain.NewResistanceProfile()
	for _, damageType := range domain.LocationDamageTypes() {
		if err := profile.SetGlobalResistance(damageType, domain.Resistance{
			Immune: inputs.globalChecked(damageType),
		}); err != nil {
			return domain.ResistanceProfile{}, err
		}
		for _, location := range domain.BodyLocations() {
			value := 0
			if !torsoOnly || location == domain.BodyTorso {
				parsed, err := parseNonNegativeIntOrError(
					inputs.locationText(damageType, location),
					resistanceFieldName(damageType, location),
					label,
				)
				if err != nil {
					return domain.ResistanceProfile{}, err
				}
				value = parsed
			}
			if err := profile.SetLocationResistance(damageType, location, value); err != nil {
				return domain.ResistanceProfile{}, err
			}
		}
	}

	poisonValue, poisonImmune, err := parseResistanceCell(
		label,
		damageTextLabel(domain.DamagePoison),
		inputs.globalText(domain.DamagePoison),
		inputs.globalChecked(domain.DamagePoison),
	)
	if err != nil {
		return domain.ResistanceProfile{}, err
	}
	if err := profile.SetGlobalResistance(domain.DamagePoison, domain.Resistance{
		Value:  poisonValue,
		Immune: poisonImmune,
	}); err != nil {
		return domain.ResistanceProfile{}, err
	}
	return profile, nil
}

func (inputs uiResistanceInputs) entries() []*widget.Entry {
	entries := make([]*widget.Entry, 0, len(domain.LocationDamageTypes())*len(domain.BodyLocations())+1)
	for _, damageType := range domain.LocationDamageTypes() {
		entries = append(entries, inputs.locationEntries(damageType)...)
	}
	if entry := inputs.globalEntry(domain.DamagePoison); entry != nil {
		entries = append(entries, entry)
	}
	return entries
}

func (inputs uiResistanceInputs) nonTorsoEntries() []*widget.Entry {
	entries := make([]*widget.Entry, 0, len(domain.LocationDamageTypes())*(len(domain.BodyLocations())-1))
	for _, damageType := range domain.LocationDamageTypes() {
		for _, location := range domain.BodyLocations() {
			if location == domain.BodyTorso {
				continue
			}
			if entry := inputs.locationEntry(damageType, location); entry != nil {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

func (inputs uiResistanceInputs) bodyGrid() *fyne.Container {
	damageTypes := domain.LocationDamageTypes()
	columns := len(damageTypes) + 1
	headerCells := []fyne.CanvasObject{newTableHeaderLabel("Body Part")}
	for _, damageType := range damageTypes {
		headerCells = append(headerCells, newTableHeaderLabel(damageTitleLabel(damageType)))
	}
	rows := []fyne.CanvasObject{container.NewGridWithColumns(columns, headerCells...)}

	immuneCells := []fyne.CanvasObject{drPartLabel("Immune")}
	for _, damageType := range damageTypes {
		immuneCells = append(immuneCells, inputs.globalImmune(damageType))
	}
	rows = append(rows, container.NewGridWithColumns(columns, immuneCells...))

	for _, location := range domain.BodyLocations() {
		cells := []fyne.CanvasObject{drPartLabel(bodyLocationTitle(location))}
		for _, damageType := range damageTypes {
			cells = append(cells, inputs.locationEntry(damageType, location))
		}
		rows = append(rows, container.NewGridWithColumns(columns, cells...))
	}

	return container.NewVBox(rows...)
}

func (inputs uiResistanceInputs) torsoGrid() *fyne.Container {
	damageTypes := domain.LocationDamageTypes()
	cells := make([]fyne.CanvasObject, 0, len(damageTypes)*2)
	for _, damageType := range damageTypes {
		cells = append(cells, newTableHeaderLabel("DR "+damageTitleLabel(damageType)))
	}
	for _, damageType := range damageTypes {
		cells = append(cells, inputs.locationEntry(damageType, domain.BodyTorso))
	}
	return container.NewGridWithColumns(len(damageTypes), cells...)
}

func (inputs uiResistanceInputs) globalEntry(damageType domain.DamageType) *widget.Entry {
	if inputs.global == nil {
		return nil
	}
	return inputs.global[damageType].entry
}

func (inputs uiResistanceInputs) globalCell(damageType domain.DamageType) fyne.CanvasObject {
	if inputs.global == nil {
		return widget.NewLabel("")
	}
	global := inputs.global[damageType]
	if global.cell != nil {
		return global.cell
	}
	if global.entry != nil {
		return global.entry
	}
	if global.immune != nil {
		return global.immune
	}
	return widget.NewLabel("")
}

func (inputs uiResistanceInputs) globalImmune(damageType domain.DamageType) *widget.Check {
	if inputs.global == nil {
		return nil
	}
	return inputs.global[damageType].immune
}

func (inputs uiResistanceInputs) locationEntry(damageType domain.DamageType, location domain.BodyLocation) *widget.Entry {
	if inputs.byLocation == nil {
		return nil
	}
	byLocation := inputs.byLocation[damageType]
	if byLocation == nil {
		return nil
	}
	return byLocation[location]
}

func (inputs uiResistanceInputs) locationEntries(damageType domain.DamageType) []*widget.Entry {
	entries := make([]*widget.Entry, 0, len(domain.BodyLocations()))
	for _, location := range domain.BodyLocations() {
		if entry := inputs.locationEntry(damageType, location); entry != nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (inputs uiResistanceInputs) globalChecked(damageType domain.DamageType) bool {
	check := inputs.globalImmune(damageType)
	return check != nil && check.Checked
}

func (inputs uiResistanceInputs) globalText(damageType domain.DamageType) string {
	if entry := inputs.globalEntry(damageType); entry != nil {
		return entry.Text
	}
	return "0"
}

func (inputs uiResistanceInputs) locationText(damageType domain.DamageType, location domain.BodyLocation) string {
	if entry := inputs.locationEntry(damageType, location); entry != nil {
		return strings.TrimSpace(entry.Text)
	}
	return "0"
}

func combatantGlobalResistance(c domain.Combatant, damageType domain.DamageType) (int, bool) {
	value, immune, err := c.GlobalResistance(damageType)
	if err != nil {
		return 0, false
	}
	return value, immune
}

func combatantLocationResistance(c domain.Combatant, damageType domain.DamageType, location domain.BodyLocation) int {
	value, err := c.LocationResistance(damageType, location)
	if err != nil {
		return 0
	}
	return value
}

func formatCombatantGlobalResistance(c domain.Combatant, damageType domain.DamageType) string {
	value, immune := combatantGlobalResistance(c, damageType)
	return formatDRValue(value, immune)
}

func formatCombatantLocationResistance(c domain.Combatant, damageType domain.DamageType, location domain.BodyLocation) string {
	_, immune := combatantGlobalResistance(c, damageType)
	if immune {
		return formatDRValue(0, true)
	}
	return strconv.Itoa(combatantLocationResistance(c, damageType, location))
}

func drPartLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	return label
}

func resistanceFieldName(damageType domain.DamageType, location domain.BodyLocation) string {
	return fmt.Sprintf("DR %s %s", damageTextLabel(damageType), bodyLocationText(location))
}

func resistancePlaceholder(damageType domain.DamageType, location domain.BodyLocation) string {
	return fmt.Sprintf("%s %s", damageAbbreviation(damageType), bodyLocationAbbreviation(location))
}

func damageAbbreviation(damageType domain.DamageType) string {
	switch damageType {
	case domain.DamagePhysical:
		return "DRP"
	case domain.DamageEnergy:
		return "DRE"
	case domain.DamageRadiation:
		return "DRR"
	default:
		return "DR"
	}
}

func damageTitleLabel(damageType domain.DamageType) string {
	switch damageType {
	case domain.DamagePhysical:
		return "Physical"
	case domain.DamageEnergy:
		return "Energy"
	case domain.DamageRadiation:
		return "Radiation"
	case domain.DamagePoison:
		return "Poison"
	default:
		return titleWords(strings.ReplaceAll(string(damageType), "_", " "))
	}
}

func damageTextLabel(damageType domain.DamageType) string {
	return strings.ReplaceAll(string(damageType), "_", " ")
}

func bodyLocationTitle(location domain.BodyLocation) string {
	switch location {
	case domain.BodyHead:
		return "Head"
	case domain.BodyTorso:
		return "Torso"
	case domain.BodyLeftArm:
		return "Left Arm"
	case domain.BodyRightArm:
		return "Right Arm"
	case domain.BodyLeftLeg:
		return "Left Leg"
	case domain.BodyRightLeg:
		return "Right Leg"
	default:
		return titleWords(bodyLocationText(location))
	}
}

func bodyLocationText(location domain.BodyLocation) string {
	return strings.ReplaceAll(string(location), "_", " ")
}

func bodyLocationAbbreviation(location domain.BodyLocation) string {
	switch location {
	case domain.BodyHead:
		return "H"
	case domain.BodyTorso:
		return "T"
	case domain.BodyLeftArm:
		return "LA"
	case domain.BodyRightArm:
		return "RA"
	case domain.BodyLeftLeg:
		return "LL"
	case domain.BodyRightLeg:
		return "RL"
	default:
		return strings.ToUpper(strings.ReplaceAll(string(location), "_", " "))
	}
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}
