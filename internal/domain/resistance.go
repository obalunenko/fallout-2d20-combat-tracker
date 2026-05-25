package domain

import "fmt"

type CombatantStats struct {
	TorsoOnly  bool
	Level      int
	XP         int
	Initiative int
	HP         int
	MaxHP      int
	Defense    int
}

type Resistance struct {
	Value  int
	Immune bool
}

type ResistanceProfile struct {
	Global     map[DamageType]Resistance
	ByLocation map[DamageType]map[BodyLocation]int
}

type CombatantProfile struct {
	Stats      CombatantStats
	Resistance ResistanceProfile
}

func NewResistanceProfile() ResistanceProfile {
	return ResistanceProfile{
		Global:     make(map[DamageType]Resistance),
		ByLocation: make(map[DamageType]map[BodyLocation]int),
	}
}

func DamageTypes() []DamageType {
	return []DamageType{
		DamagePhysical,
		DamageEnergy,
		DamageRadiation,
		DamagePoison,
	}
}

func LocationDamageTypes() []DamageType {
	return []DamageType{
		DamagePhysical,
		DamageEnergy,
		DamageRadiation,
	}
}

func BodyLocations() []BodyLocation {
	return []BodyLocation{
		BodyHead,
		BodyTorso,
		BodyLeftArm,
		BodyRightArm,
		BodyLeftLeg,
		BodyRightLeg,
	}
}

func (c Combatant) Stats() CombatantStats {
	return CombatantStats{
		TorsoOnly:  c.TorsoOnly,
		Level:      c.Level,
		XP:         c.XP,
		Initiative: c.Initiative,
		HP:         c.HP,
		MaxHP:      c.MaxHP,
		Defense:    c.Defense,
	}
}

func (c *Combatant) SetStats(stats CombatantStats) {
	if c == nil {
		return
	}
	c.TorsoOnly = stats.TorsoOnly
	c.Level = stats.Level
	c.XP = stats.XP
	c.Initiative = stats.Initiative
	c.HP = stats.HP
	c.MaxHP = stats.MaxHP
	c.Defense = stats.Defense
}

func (c Combatant) Profile() CombatantProfile {
	return CombatantProfile{
		Stats:      c.Stats(),
		Resistance: c.ResistanceProfile(),
	}
}

func (c *Combatant) SetProfile(profile CombatantProfile) {
	if c == nil {
		return
	}
	c.SetStats(profile.Stats)
	c.SetResistanceProfile(profile.Resistance)
}

func (c Combatant) ResistanceProfile() ResistanceProfile {
	return c.resistanceProfile(true)
}

func (c Combatant) GlobalResistance(damageType DamageType) (int, bool, error) {
	return c.ResistanceProfile().GlobalResistance(damageType)
}

func (c Combatant) LocationResistance(damageType DamageType, location BodyLocation) (int, error) {
	return c.ResistanceProfile().LocationResistance(damageType, location)
}

func (c *Combatant) SetGlobalResistance(damageType DamageType, value int, immune bool) error {
	if c == nil {
		return nil
	}
	profile := c.ResistanceProfile()
	if err := profile.SetGlobalResistance(damageType, Resistance{Value: value, Immune: immune}); err != nil {
		return err
	}
	c.SetResistanceProfile(profile)
	return nil
}

func (c *Combatant) SetLocationResistance(damageType DamageType, location BodyLocation, value int) error {
	if c == nil {
		return nil
	}
	profile := c.ResistanceProfile()
	if err := profile.SetLocationResistance(damageType, location, value); err != nil {
		return err
	}
	c.SetResistanceProfile(profile)
	return nil
}

func (c Combatant) damageResistance(damageType DamageType, location BodyLocation) (int, bool, error) {
	return c.ResistanceProfile().EffectiveResistance(damageType, location, c.TorsoOnly)
}

func (c Combatant) HasNegativeResistance() bool {
	return c.resistanceProfile(false).HasNegativeValues()
}

func (p ResistanceProfile) HasNegativeValues() bool {
	if p.Global != nil && p.Global[DamagePoison].Value < 0 {
		return true
	}
	for _, damageType := range LocationDamageTypes() {
		if p.ByLocation == nil || p.ByLocation[damageType] == nil {
			continue
		}
		for _, location := range BodyLocations() {
			if p.ByLocation[damageType][location] < 0 {
				return true
			}
		}
	}
	return false
}

func (c Combatant) resistanceProfile(normalize bool) ResistanceProfile {
	return ResistanceProfile{
		Global: map[DamageType]Resistance{
			DamagePhysical:  {Immune: c.ImmunePhysical},
			DamageEnergy:    {Immune: c.ImmuneEnergy},
			DamageRadiation: {Immune: c.ImmuneRadiation},
			DamagePoison:    {Value: resistanceValue(c.ResistPoison, normalize), Immune: c.ImmunePoison},
		},
		ByLocation: map[DamageType]map[BodyLocation]int{
			DamagePhysical: {
				BodyHead:     resistanceValue(c.ResistPhysicalHead, normalize),
				BodyTorso:    resistanceValue(c.ResistPhysicalTorso, normalize),
				BodyLeftArm:  resistanceValue(c.ResistPhysicalLeftArm, normalize),
				BodyRightArm: resistanceValue(c.ResistPhysicalRightArm, normalize),
				BodyLeftLeg:  resistanceValue(c.ResistPhysicalLeftLeg, normalize),
				BodyRightLeg: resistanceValue(c.ResistPhysicalRightLeg, normalize),
			},
			DamageEnergy: {
				BodyHead:     resistanceValue(c.ResistEnergyHead, normalize),
				BodyTorso:    resistanceValue(c.ResistEnergyTorso, normalize),
				BodyLeftArm:  resistanceValue(c.ResistEnergyLeftArm, normalize),
				BodyRightArm: resistanceValue(c.ResistEnergyRightArm, normalize),
				BodyLeftLeg:  resistanceValue(c.ResistEnergyLeftLeg, normalize),
				BodyRightLeg: resistanceValue(c.ResistEnergyRightLeg, normalize),
			},
			DamageRadiation: {
				BodyHead:     resistanceValue(c.ResistRadiationHead, normalize),
				BodyTorso:    resistanceValue(c.ResistRadiationTorso, normalize),
				BodyLeftArm:  resistanceValue(c.ResistRadiationLeftArm, normalize),
				BodyRightArm: resistanceValue(c.ResistRadiationRightArm, normalize),
				BodyLeftLeg:  resistanceValue(c.ResistRadiationLeftLeg, normalize),
				BodyRightLeg: resistanceValue(c.ResistRadiationRightLeg, normalize),
			},
		},
	}
}

func resistanceValue(value int, normalize bool) int {
	if normalize {
		return max(value, 0)
	}
	return value
}

func (c *Combatant) SetResistanceProfile(profile ResistanceProfile) {
	if c == nil {
		return
	}

	_, c.ImmunePhysical = profile.globalValue(DamagePhysical)
	_, c.ImmuneEnergy = profile.globalValue(DamageEnergy)
	_, c.ImmuneRadiation = profile.globalValue(DamageRadiation)
	c.ResistPhysical = 0
	c.ResistEnergy = 0
	c.ResistRadiation = 0
	c.ResistPoison, c.ImmunePoison = profile.globalValue(DamagePoison)

	c.ResistPhysicalHead = profile.locationValue(DamagePhysical, BodyHead)
	c.ResistPhysicalTorso = profile.locationValue(DamagePhysical, BodyTorso)
	c.ResistPhysicalLeftArm = profile.locationValue(DamagePhysical, BodyLeftArm)
	c.ResistPhysicalRightArm = profile.locationValue(DamagePhysical, BodyRightArm)
	c.ResistPhysicalLeftLeg = profile.locationValue(DamagePhysical, BodyLeftLeg)
	c.ResistPhysicalRightLeg = profile.locationValue(DamagePhysical, BodyRightLeg)

	c.ResistEnergyHead = profile.locationValue(DamageEnergy, BodyHead)
	c.ResistEnergyTorso = profile.locationValue(DamageEnergy, BodyTorso)
	c.ResistEnergyLeftArm = profile.locationValue(DamageEnergy, BodyLeftArm)
	c.ResistEnergyRightArm = profile.locationValue(DamageEnergy, BodyRightArm)
	c.ResistEnergyLeftLeg = profile.locationValue(DamageEnergy, BodyLeftLeg)
	c.ResistEnergyRightLeg = profile.locationValue(DamageEnergy, BodyRightLeg)

	c.ResistRadiationHead = profile.locationValue(DamageRadiation, BodyHead)
	c.ResistRadiationTorso = profile.locationValue(DamageRadiation, BodyTorso)
	c.ResistRadiationLeftArm = profile.locationValue(DamageRadiation, BodyLeftArm)
	c.ResistRadiationRightArm = profile.locationValue(DamageRadiation, BodyRightArm)
	c.ResistRadiationLeftLeg = profile.locationValue(DamageRadiation, BodyLeftLeg)
	c.ResistRadiationRightLeg = profile.locationValue(DamageRadiation, BodyRightLeg)
}

func (p ResistanceProfile) GlobalResistance(damageType DamageType) (int, bool, error) {
	if !isKnownDamageType(damageType) {
		return 0, false, fmt.Errorf("unknown damage type: %q", damageType)
	}
	value, immune := p.globalValue(damageType)
	if damageType != DamagePoison {
		return 0, immune, nil
	}
	return value, immune, nil
}

func (p *ResistanceProfile) SetGlobalResistance(damageType DamageType, resistance Resistance) error {
	if !isKnownDamageType(damageType) {
		return fmt.Errorf("unknown damage type: %q", damageType)
	}
	if p.Global == nil {
		p.Global = make(map[DamageType]Resistance)
	}
	if damageType != DamagePoison {
		resistance.Value = 0
	}
	p.Global[damageType] = resistance
	return nil
}

func (p ResistanceProfile) LocationResistance(damageType DamageType, location BodyLocation) (int, error) {
	if !isKnownDamageType(damageType) {
		return 0, fmt.Errorf("unknown damage type: %q", damageType)
	}
	if !isKnownBodyLocation(location) {
		return 0, fmt.Errorf("unknown body location: %q", location)
	}
	if damageType == DamagePoison {
		return 0, nil
	}
	return p.locationValue(damageType, location), nil
}

func (p *ResistanceProfile) SetLocationResistance(damageType DamageType, location BodyLocation, value int) error {
	if !isKnownDamageType(damageType) {
		return fmt.Errorf("unknown damage type: %q", damageType)
	}
	if !isKnownBodyLocation(location) {
		return fmt.Errorf("unknown body location: %q", location)
	}
	if damageType == DamagePoison {
		return fmt.Errorf("poison resistance is global-only")
	}
	if p.ByLocation == nil {
		p.ByLocation = make(map[DamageType]map[BodyLocation]int)
	}
	byLocation := p.ByLocation[damageType]
	if byLocation == nil {
		byLocation = make(map[BodyLocation]int)
		p.ByLocation[damageType] = byLocation
	}
	byLocation[location] = value
	return nil
}

func (p ResistanceProfile) EffectiveResistance(damageType DamageType, location BodyLocation, torsoOnly bool) (int, bool, error) {
	poisonResistance, immune, err := p.GlobalResistance(damageType)
	if err != nil {
		return 0, false, err
	}
	if damageType == DamagePoison {
		return poisonResistance, immune, nil
	}
	if !isKnownBodyLocation(location) {
		return 0, false, fmt.Errorf("unknown body location: %q", location)
	}
	if torsoOnly {
		location = BodyTorso
	}
	locationResistance, err := p.LocationResistance(damageType, location)
	if err != nil {
		return 0, false, err
	}
	return locationResistance, immune, nil
}

func (p ResistanceProfile) Clone() ResistanceProfile {
	clone := NewResistanceProfile()
	for damageType, resistance := range p.Global {
		clone.Global[damageType] = resistance
	}
	for damageType, byLocation := range p.ByLocation {
		cloneByLocation := make(map[BodyLocation]int, len(byLocation))
		for location, resistance := range byLocation {
			cloneByLocation[location] = resistance
		}
		clone.ByLocation[damageType] = cloneByLocation
	}
	return clone
}

func (p ResistanceProfile) globalValue(damageType DamageType) (int, bool) {
	if p.Global == nil {
		return 0, false
	}
	resistance := p.Global[damageType]
	return max(resistance.Value, 0), resistance.Immune
}

func (p ResistanceProfile) locationValue(damageType DamageType, location BodyLocation) int {
	if p.ByLocation == nil {
		return 0
	}
	byLocation := p.ByLocation[damageType]
	if byLocation == nil {
		return 0
	}
	return max(byLocation[location], 0)
}

func isKnownDamageType(damageType DamageType) bool {
	for _, known := range DamageTypes() {
		if damageType == known {
			return true
		}
	}
	return false
}

func isKnownBodyLocation(location BodyLocation) bool {
	for _, known := range BodyLocations() {
		if location == known {
			return true
		}
	}
	return false
}
