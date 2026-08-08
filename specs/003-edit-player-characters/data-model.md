# Data Model: Edit Campaign Player Characters

## Player Character Details

Represents player-character-only information carried with an existing campaign roster entry.

**Fields**:

- `player character ID`: stable identity linking the details to one active or historical player character.
- `notes`: free-form multiline text; blank is valid and whitespace is preserved exactly.
- `S.P.E.C.I.A.L.`: one value for each of Strength, Perception, Endurance, Charisma, Intelligence, Agility, and Luck.
- `combat profile`: the existing level, initiative, current HP, maximum HP, Defense, torso mode, DR, and immunity state.
- `availability`: the existing active/inactive roster status.

**Relationships**:

- Belongs to one player, whose player record belongs to one campaign.
- Owns exactly one logical value for every defined S.P.E.C.I.A.L. attribute.
- Uses the existing player-character stat profile and normalized resistance rows for combat values.
- May be linked from combatants in one or more campaign encounters.

**Validation rules**:

- Notes may be empty and are not trimmed.
- Level is at least 1.
- Every S.P.E.C.I.A.L. value is a whole number of at least 1, with no feature-level upper bound.
- Current HP is at least 0; maximum HP is at least 1; current HP does not exceed maximum HP.
- Defense and DR values are at least 0.
- Physical, energy, and radiation DR are body-location-specific; poison DR is global.
- Existing immunity values remain valid and are preserved.

## S.P.E.C.I.A.L. Attribute Definition

Dictionary entity defining the supported attribute dimension.

**Fields**:

- `id`: stable numeric dictionary key.
- `code`: one of `strength`, `perception`, `endurance`, `charisma`, `intelligence`, `agility`, or `luck`.

**Relationships**:

- Referenced by player-character attribute values.
- Seven stable rows are seeded by migration and are not user-created.

**Validation rules**:

- Code is unique and restricted to the seven supported values.
- Dictionary identifiers and codes are stable across application runs.

## Player Character S.P.E.C.I.A.L. Value

Normalized assignment of one S.P.E.C.I.A.L. value to one player character.

**Fields**:

- `player character ID`: owner reference.
- `special attribute ID`: dictionary reference.
- `value`: positive whole number.
- `created at`, `updated at`: persistence timestamps.

**Relationships**:

- Composite identity is `(player character ID, special attribute ID)`.
- Deleted automatically when its player character is deleted.
- Exactly seven logical values are assembled into the domain value object; migration and repository writes ensure all seven are present.

**Validation rules**:

- Value is at least 1.
- Duplicate assignments for one character/attribute pair are impossible.
- Missing rows found in legacy or damaged data are reported rather than silently producing a partially populated edit form; the supported migration backfills valid legacy databases.

## Encounter Combatant Snapshot

Represents encounter-specific combat state stored independently from the campaign player-character profile.

**Fields**:

- Existing combatant identity, player-character link, side, position, defeated state, and stat profile.
- Existing scalar combat values and normalized resistance profile.

**Relationships**:

- Optionally references a player character in the same campaign.
- Receives current campaign combat values when first added to an encounter.
- Receives new campaign combat values when it belongs to the effective active encounter during a successful campaign edit.
- Does not receive notes or S.P.E.C.I.A.L. values.

**Validation rules**:

- A linked combatant must belong to an encounter in the same campaign as its player character.
- Active-sync current HP determines defeated state: 0 is defeated; above 0 is not defeated.
- Active sync preserves combatant identity, player-character link, position, encounter round, turn index, and all unrelated combatants.
- Closed encounter snapshots remain unchanged by campaign edits.

## Migration Defaults

- Existing player-character notes become an empty string.
- Existing player characters receive all seven S.P.E.C.I.A.L. rows with value 1.
- New player characters receive the same seven defaults when no explicit values are supplied during campaign creation.
- Existing scalar stats, DR, immunity, identity, availability, and encounter snapshots are retained.

## State Transitions

1. **Open campaign editor** → Stored notes, S.P.E.C.I.A.L., combat profile, and availability populate each character row and its expandable details.
2. **Edit draft** → Values exist only in widget state; persisted character and encounter data are unchanged.
3. **Attempt invalid Save** → Domain/application validation rejects the draft; editor remains open with entered values; no repository mutation occurs.
4. **Cancel** → Editor closes; all draft changes are discarded.
5. **Save valid edit without an active linked combatant** → Player-character details and combat profile commit; campaign views refresh; encounter snapshots remain unchanged.
6. **Save valid edit with an active linked combatant** → Player-character data and that combatant’s combat profile/defeated state commit in one transaction; active views refresh.
7. **Persistence failure** → Transaction rolls back player-character and combatant mutations together; editor remains open with an actionable error.
8. **Reopen application** → Notes and all seven S.P.E.C.I.A.L. values load unchanged from persistent storage.
