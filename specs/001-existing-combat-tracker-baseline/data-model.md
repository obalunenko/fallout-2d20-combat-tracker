# Data Model: Existing Combat Tracker Baseline

## Campaign

Represents a tabletop campaign context.

**Fields**: ID, name, start date, Party AP, GM Threat, updated timestamp.

**Relationships**:

- Owns players and player characters.
- Owns encounters.
- Supplies shared resources to active encounters.

**Validation**:

- Name is required.
- Start date is required and uses `YYYY-MM-DD` at UI input boundaries.
- At least one player is required when creating/updating through the current UI/service flow.

## Player

Represents a real player within a campaign.

**Fields**: ID, campaign ID, name.

**Relationships**:

- Owns one or more historical player characters.
- Has at most one active current character per campaign flow.

**Validation**:

- Player name is required.
- Duplicate active players by normalized name are rejected during campaign update.

## PlayerCharacter

Represents a party combatant profile attached to a player.

**Fields**: ID, player ID, name, level, initiative, HP, max HP, defense, active/inactive availability, stat profile, resistance profile.

**Relationships**:

- May be linked into encounter combatants.
- Updates from linked party combatants can synchronize back to campaign character state.

**Validation**:

- Character name is required.
- Level is at least 1 for campaign party characters.
- HP, max HP, initiative, defense, and resistance values are non-negative.
- Current HP cannot exceed max HP.
- Side is normalized to `party`; XP is normalized to 0.

**State transitions**:

- Active to inactive when removed from roster or explicitly marked inactive.
- Existing character to historical inactive character when a player's active character name changes.
- Alive to defeated when HP reaches 0; defeated to alive when healing restores HP above 0.

## Encounter

Represents a combat instance within a campaign.

**Fields**: ID, campaign ID, name, round, turn index, resources snapshot/reference, combatants.

**Relationships**:

- Belongs to one campaign.
- Contains ordered combatants.
- Owns encounter logs.

**Validation**:

- Name is required.
- At least one combatant is required for create/update.
- Round normalizes to at least 1.
- Turn index normalizes within combatant bounds.
- Duplicate linked party characters in the same encounter are rejected.

**State transitions**:

- Created with round 1 and first sorted combatant active.
- Activated as the active campaign encounter.
- Updated while preserving current turn where possible.
- Restarted by resetting NPC HP/defeated state and preserving campaign resources.
- Soft-deleted so it no longer appears in active lists.

## Combatant

Represents a participant in an encounter or a reusable monster template.

**Fields**: ID, optional player character ID, name, side, torso-only flag, level, XP, initiative, HP, max HP, defense, resistance profile, immunity flags, active flag, defeated flag.

**Relationships**:

- May link to a player character when side is `party`.
- May originate from a monster template when side is `npc`.

**Validation**:

- Name is required for encounter and template flows.
- Side is `party` or `npc` when required.
- Numeric stats and resistance values are non-negative.
- Current HP cannot exceed max HP.
- Party combatants have XP normalized to 0 in campaign contexts.

**State transitions**:

- Active flag follows encounter turn index.
- HP decreases by effective damage and sets defeated at 0.
- HP increases by healing up to max HP and clears defeated when above 0.

## Resources

Represents Party AP and GM Threat.

**Fields**: Party AP, GM Threat.

**Validation**:

- Party AP cannot exceed 6 and cannot go below 0.
- GM Threat cannot go below 0.
- Spending more than available fails.

## ResistanceProfile

Represents typed damage resistance and immunity.

**Fields**:

- Global resistance map keyed by damage type.
- Location resistance map keyed by damage type and body location.

**Rules**:

- Damage types: physical, energy, radiation, poison.
- Body locations: head, torso, left arm, right arm, left leg, right leg.
- Physical, energy, and radiation use location resistance.
- Poison uses global resistance only.
- Immunity reduces effective damage to zero.
- Torso-only combatants resolve location damage as torso damage.

## MonsterTemplate

Represents a reusable NPC combatant profile.

**Fields**: ID, name, level, XP, initiative, HP, max HP, defense, torso-only flag, resistance profile, immunity flags.

**Relationships**:

- Can be loaded into encounter rows as a fresh NPC combatant.

**Validation**:

- Name is required.
- Level is at least 1.
- Side is normalized to `npc`.
- Player character link, active state, and defeated state are cleared on save.
- Templates are upserted by existing name.

## EncounterLog

Represents a round-stamped operation message for the DATA view.

**Fields**: ID, encounter ID, round, message, created timestamp.

**Relationships**:

- Belongs to one encounter.

**Validation**:

- Encounter ID is required.
- Message is required.

**Failure behavior**:

- Append failure is a non-critical side effect for primary combat/resource operations.

## StatProfile And Normalized Resistance Rows

Represent normalized storage for scalar combat stats and resistance profiles across combatants, player characters, and monster templates.

**Relationships**:

- Referenced by persisted combatant-like entities.
- Use dictionary tables for body locations and damage types.

**Validation**:

- Dictionary values define allowed locations and damage types.
- Future stat dimensions should use row-based extensions where possible.
