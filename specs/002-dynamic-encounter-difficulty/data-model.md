# Data Model: Dynamic Encounter Difficulty Calculator

## Encounter Draft

Represents the unsaved encounter composition currently visible in the create/edit dialog.

**Fields**:

- `name`: encounter draft name, owned by existing editor validation.
- `combatant rows`: ordered draft rows containing party or monster entries.
- `selected player characters`: rows whose side is party.
- `monster entries`: rows whose side is NPC/monster.

**Relationships**:

- Selected player characters may be loaded from the campaign roster.
- Monster entries may be manually entered or loaded from monster templates.
- Draft rows become persisted encounter combatants only when the GM saves.

**Validation rules**:

- At least one selected player is required before difficulty can be calculated.
- Player level must be a valid positive integer for preview calculation.
- Monster XP must be a valid non-negative integer for preview calculation.
- Monster quantity must be a valid positive integer for preview calculation.
- Invalid or incomplete calculation fields produce an unavailable difficulty preview, not a stale label.

## Selected Player Character

Represents a party-side draft participant used for the difficulty formula.

**Fields**:

- `level`: positive integer contributing to Average PC Level.

**Relationships**:

- Count of selected player characters is the denominator for XP Baseline.

**Validation rules**:

- Level must be positive.
- Each selected player counts once, regardless of any draft quantity field.

## Monster Entry

Represents a non-party draft participant used for the difficulty formula.

**Fields**:

- `xp`: non-negative integer XP value for one monster.
- `quantity`: positive integer count of monsters represented by the row.

**Relationships**:

- Total Monster XP sums `xp * quantity` across all monster entries.
- Monster entries loaded from templates use the template XP unless edited before save.

**Validation rules**:

- XP must be non-negative.
- Quantity must be at least 1.

## Difficulty Result

Represents the calculated preview for the current draft.

**Fields**:

- `label`: one of Unknown, Trivial, Simple, Average, Hard, Deadly.
- `party count`: number of selected player characters.
- `average PC level`: `ceil(sum(player levels) / party count)`.
- `total monster XP`: sum of all monster XP contributions.
- `XP baseline`: `total monster XP / party count`.
- `encounter level`: `floor((XP baseline - 10) / 10)`, with a minimum of 1.
- `difference`: `encounter level - average PC level`.
- `unavailable reason`: display-safe reason when calculation cannot produce a label.

**Label rules**:

- Difference less than -2: Trivial.
- Difference -2 or -1: Simple.
- Difference 0 or 1: Average.
- Difference 2 through 5: Hard.
- Difference greater than 5: Deadly.

## State Transitions

1. **No valid players** -> Difficulty Result is Unknown/unavailable.
2. **Players valid, monster data valid** -> Difficulty Result has a calculated label.
3. **Draft numeric field becomes invalid** -> Difficulty Result returns to Unknown/unavailable.
4. **Draft numeric field is corrected** -> Difficulty Result recalculates immediately.
5. **GM saves encounter** -> Existing save flow persists combatants; difficulty remains derived.
6. **GM cancels editor** -> No encounter data is changed because preview state is not persisted.
