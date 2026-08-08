# UI/Application Contract: Edit Campaign Player Characters

## Surface

The existing Create/Edit Campaign dialog keeps its compact roster rows and adds an expandable Character Details section for each player character. The outer campaign Save remains the only persistence action; Cancel discards all draft changes.

## Prefill Contract

For an existing character, the editor displays:

- player and character identity fields already supported by the campaign editor;
- level, initiative, current HP, maximum HP, Defense, availability, and the existing DR/immunity profile;
- multiline notes, including exact whitespace and line breaks;
- Strength, Perception, Endurance, Charisma, Intelligence, Agility, and Luck.

For a newly added character, notes are blank and all seven S.P.E.C.I.A.L. fields default to 1. Existing defaults for combat fields remain in effect.

## Input And Validation Contract

- Notes accept blank or multiline free-form text and are not trimmed.
- Level and each S.P.E.C.I.A.L. field accept whole numbers of at least 1.
- Current HP accepts a whole number of at least 0.
- Maximum HP accepts a whole number of at least 1.
- Current HP must not exceed maximum HP.
- Defense and every DR value accept whole numbers of at least 0.
- The existing resistance collector preserves global immunity values and enforces body-location/poison rules.
- Validation feedback identifies the player/character and invalid field.
- A validation error leaves the dialog open and does not clear any entered field.

## Save Contract

1. Collect every campaign row, including collapsed character-detail fields.
2. Validate campaign identity, roster identity, character details, and combat profiles before repository mutation.
3. Submit the entire campaign edit as one application operation.
4. Persist player-character notes, all seven S.P.E.C.I.A.L. values, existing combat stats, DR, immunity, and availability in one transaction.
5. If the edited campaign is active and its effective active encounter contains a linked character, copy level, current HP, maximum HP, Defense, and the complete DR/immunity profile to that combatant snapshot and derive defeated state from current HP.
6. Preserve active combatant identity, position, encounter round and turn, encounter resources, and unrelated combatants.
7. Commit all campaign and active-encounter changes together.
8. Close the dialog and invoke the existing full refresh callback only after success.

## Cancel Contract

- Cancel closes the editor without calling campaign persistence.
- No character, campaign, or encounter value changes.
- Reopening the editor displays the previously stored values.

## Encounter Snapshot Contract

- The effective active encounter is the most recently activated/updated non-deleted encounter for the application’s active campaign, matching existing activation behavior.
- A campaign edit synchronizes linked combatant values only when editing that active campaign.
- Closed encounters read their own stored combatant scalar and resistance profiles; they do not substitute the latest campaign profile.
- Notes and S.P.E.C.I.A.L. are never copied into encounter combatants.
- Explicit encounter saves may continue synchronizing linked party combat state back to the campaign according to existing behavior.

## Refresh Contract

After successful Save, refresh:

- active campaign overview;
- active roster output;
- party library;
- active encounter state, encounter order, active-target details, tactical snapshot, and defeated indicators when an active linked combatant changed.

On validation or persistence failure, keep the current editor state visible and leave previously rendered main-screen state unchanged except for the error message.

## Error Contract

- Missing campaign or character identity returns a not-found or required-field error and does not create a replacement.
- Invalid numeric or resistance input returns field-specific feedback before persistence.
- Missing S.P.E.C.I.A.L. rows in a migrated database are treated as inconsistent data and reported rather than silently discarding part of a character profile.
- Any database failure rolls back both campaign character and active-combatant mutations.
- Audit-log behavior is unchanged and is not a prerequisite for a successful character edit.
