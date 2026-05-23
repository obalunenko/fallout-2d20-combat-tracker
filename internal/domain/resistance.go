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

func (c Combatant) ResistanceProfile() ResistanceProfile {
	return c.resistanceProfile(true)
}

func (c Combatant) damageResistance(damageType DamageType, location BodyLocation) (int, bool, error) {
	profile := c.ResistanceProfile()
	poisonResistance, immune, err := profile.GlobalResistance(damageType)
	if err != nil {
		return 0, false, err
	}
	if damageType == DamagePoison {
		return poisonResistance, immune, nil
	}
	if !isKnownBodyLocation(location) {
		return 0, false, fmt.Errorf("unknown body location: %q", location)
	}
	if c.TorsoOnly {
		torsoResistance, err := profile.LocationResistance(damageType, BodyTorso)
		if err != nil {
			return 0, false, err
		}
		return torsoResistance, immune, nil
	}
	locationResistance, err := profile.LocationResistance(damageType, location)
	if err != nil {
		return 0, false, err
	}
	return locationResistance, immune, nil
}

func (c Combatant) HasNegativeResistance() bool {
	profile := c.resistanceProfile(false)
	if profile.Global[DamagePoison].Value < 0 {
		return true
	}
	for _, damageType := range LocationDamageTypes() {
		for _, location := range BodyLocations() {
			if profile.ByLocation[damageType][location] < 0 {
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
	return value, immune, nil
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
