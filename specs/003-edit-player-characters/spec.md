# Feature Specification: Edit Campaign Player Characters

**Feature Branch**: Not created (no `before_specify` branch hook is configured)

**Created**: 2026-08-08

**Status**: Draft

**Input**: User description: "Implement feature for editing player characters in campaign. It should be possible to add notes, update level, SPECIAL values, current and max HP, defence and DR."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Edit A Character Profile (Priority: P1)

A GM can select an existing player character in a campaign and edit the character's notes, level, seven S.P.E.C.I.A.L. attributes, current and maximum HP, Defense, and damage resistance so the campaign roster reflects the character's current state.

**Why this priority**: Keeping the campaign's player-character records accurate is the core value of the feature and prevents the GM from maintaining these details outside the tracker.

**Independent Test**: Open a campaign with an existing player character, change every in-scope field, save, reopen the character, and verify every saved value is shown unchanged.

**Acceptance Scenarios**:

1. **Given** an existing campaign player character, **When** the GM opens that character for editing, **Then** the editor shows the character's current notes, level, Strength, Perception, Endurance, Charisma, Intelligence, Agility, Luck, current HP, maximum HP, Defense, and DR values.
2. **Given** valid changes to one or more character fields, **When** the GM saves, **Then** all submitted changes are stored together and the campaign roster and character details immediately show the updated values.
3. **Given** a character with multiline notes, **When** the GM saves and later reopens the character, **Then** the note content and line breaks are preserved.
4. **Given** an open character editor with unsaved changes, **When** the GM cancels, **Then** none of those changes are stored or shown in the campaign roster.

---

### User Story 2 - Prevent Invalid Character Stats (Priority: P1)

A GM receives clear feedback when character values are invalid, so an edit cannot leave the character in an unusable combat state.

**Why this priority**: Invalid HP, level, S.P.E.C.I.A.L., Defense, or DR values could break encounter preparation and combat calculations.

**Independent Test**: Attempt to save each invalid boundary condition and verify the edit remains open, identifies the problem, and leaves the stored character unchanged.

**Acceptance Scenarios**:

1. **Given** an edit where current HP is greater than maximum HP, **When** the GM saves, **Then** the save is rejected with a field-specific message and the stored character is unchanged.
2. **Given** an edit where level or any S.P.E.C.I.A.L. attribute is not a positive whole number, **When** the GM saves, **Then** the save is rejected and identifies the invalid field.
3. **Given** an edit where current HP, Defense, or any DR value is negative, or maximum HP is less than 1, **When** the GM saves, **Then** the save is rejected and identifies the invalid field.
4. **Given** multiple invalid fields, **When** the GM attempts to save, **Then** the GM receives enough field-specific feedback to correct the submission without losing the entered values.

---

### User Story 3 - Keep Active Combat In Sync (Priority: P2)

A GM editing a player character who is participating in the active encounter sees the relevant combat state update consistently, avoiding different values in the campaign roster and live combat view.

**Why this priority**: Campaign editing is valuable on its own, but conflicting values during an active encounter would create immediate confusion at the table.

**Independent Test**: Add a campaign player character to the active encounter, edit the character's combat-relevant fields from the campaign, save, and verify both the roster and active encounter show the same values while a closed encounter remains unchanged.

**Acceptance Scenarios**:

1. **Given** the edited player character is linked to the active encounter, **When** the GM saves valid changes, **Then** the linked active combatant's level, current HP, maximum HP, Defense, and DR match the saved campaign character.
2. **Given** an active linked combatant whose saved current HP becomes 0, **When** the edit succeeds, **Then** the active encounter shows that combatant as defeated; when saved HP is above 0, the combatant is not defeated.
3. **Given** the edited character appears in a closed or inactive encounter, **When** the GM saves changes from the campaign, **Then** that historical encounter remains unchanged.
4. **Given** notes or S.P.E.C.I.A.L. values are edited, **When** the GM saves, **Then** those values remain part of the campaign character profile and are not required to be copied into encounter combatant records.

### Edge Cases

- A character may have blank notes; clearing existing notes is a valid edit.
- Leading and trailing whitespace around notes is preserved because it may be intentional, while line breaks must always be preserved.
- A character's level and each S.P.E.C.I.A.L. value have a minimum of 1; this feature does not impose an upper limit so advanced characters and temporary campaign-specific values are not rejected.
- Current HP may be 0, maximum HP must be at least 1, and current HP may not exceed maximum HP.
- Defense and every DR value may be 0 but may not be negative.
- Physical, energy, and radiation DR remain body-location-specific; poison DR remains global, matching the existing tracker model.
- Existing resistance immunity settings are preserved when other character fields are edited; changing immunity is outside this feature's required scope.
- If saving fails for any reason, no partial character or active-encounter changes are retained, the editor remains open, and the GM sees an actionable error.
- If the selected character or campaign no longer exists when Save is chosen, the edit is rejected and the campaign view is refreshed without creating a replacement character.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a GM to open an existing player character for editing from its campaign context.
- **FR-002**: The editor MUST display the character's currently stored values before the GM makes changes.
- **FR-003**: The GM MUST be able to add, replace, or clear free-form multiline character notes.
- **FR-004**: The GM MUST be able to update the character's level.
- **FR-005**: The GM MUST be able to update all seven S.P.E.C.I.A.L. attributes: Strength, Perception, Endurance, Charisma, Intelligence, Agility, and Luck.
- **FR-006**: The GM MUST be able to update current HP and maximum HP independently, subject to their validation relationship.
- **FR-007**: The GM MUST be able to update Defense.
- **FR-008**: The GM MUST be able to update the existing DR profile: physical, energy, and radiation DR by body location, plus global poison DR.
- **FR-009**: Level and every S.P.E.C.I.A.L. attribute MUST be positive whole numbers.
- **FR-010**: Current HP MUST be a non-negative whole number, maximum HP MUST be a positive whole number, and current HP MUST NOT exceed maximum HP.
- **FR-011**: Defense and every DR value MUST be non-negative whole numbers.
- **FR-012**: Invalid edits MUST NOT change stored character or encounter state and MUST produce field-specific feedback without discarding the GM's entered values.
- **FR-013**: Saving a valid edit MUST persist all changed character values as one operation so either every submitted change succeeds or none does.
- **FR-014**: Canceling an edit MUST leave all stored values unchanged.
- **FR-015**: After a successful save, the system MUST refresh the campaign roster, character details, and character-selection views that use campaign player-character data.
- **FR-016**: When the edited character is linked to the active encounter, a successful save MUST synchronize level, current HP, maximum HP, Defense, and DR to that linked combatant and refresh active encounter views.
- **FR-017**: Active-encounter synchronization MUST set the linked combatant's defeated state according to whether current HP is 0 and MUST preserve the encounter's round, turn, and unrelated combatants.
- **FR-018**: Saving a campaign character edit MUST NOT rewrite combatants in closed or inactive encounters.
- **FR-019**: Notes and S.P.E.C.I.A.L. values MUST remain associated with the player character across application restarts and MUST NOT be duplicated into encounter combatant data solely for this feature.
- **FR-020**: Existing player identity, character identity, name, initiative, availability, DR immunity settings, and other fields outside the submitted edit MUST be preserved.

### Key Entities *(include if feature involves data)*

- **Player Character**: A campaign-owned character associated with a player; owns notes, level, seven S.P.E.C.I.A.L. values, HP, Defense, and a DR profile in addition to existing identity and availability data.
- **S.P.E.C.I.A.L. Profile**: The character's Strength, Perception, Endurance, Charisma, Intelligence, Agility, and Luck values, each represented as a positive whole number.
- **Damage Resistance Profile**: The existing non-negative resistance values for physical, energy, and radiation damage by body location and poison damage globally; existing immunity settings are retained.
- **Linked Active Combatant**: The active encounter participant associated with the edited player character; receives the saved combat-relevant values but not profile-only notes or S.P.E.C.I.A.L. values.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A GM can open a character, change at least one value in every in-scope field group, and save the complete edit in under 2 minutes.
- **SC-002**: In acceptance testing, 100% of valid edited values—including multiline notes and all seven S.P.E.C.I.A.L. attributes—are identical after the application is closed and reopened.
- **SC-003**: In acceptance testing, 100% of invalid boundary submissions are rejected without changing the previously stored character or encounter state.
- **SC-004**: For a campaign roster of up to 12 player characters, a successful edit is reflected in all affected on-screen campaign and active-encounter views within 2 seconds.
- **SC-005**: At least 90% of representative users can complete a valid character edit on their first attempt without external instructions.
- **SC-006**: A successful edit of a character linked to the active encounter produces no mismatches in level, HP, Defense, or DR between the campaign character and that linked combatant.

## Assumptions

- The GM is the only local user; authentication, permissions, concurrent editors, network sync, and multi-device conflict resolution remain out of scope.
- Editing occurs within the existing campaign-management experience and applies to an existing player character; creating characters or managing multiple simultaneous active characters for one player is not expanded by this feature.
- S.P.E.C.I.A.L. means Strength, Perception, Endurance, Charisma, Intelligence, Agility, and Luck. Each accepts a positive whole number, with no feature-level upper cap.
- “DR” uses the tracker's existing damage-resistance structure: body-location values for physical, energy, and radiation damage and a global poison value.
- Existing immunity controls and values continue to behave as they do today, but adding a new immunity-editing capability is not required.
- The campaign player character is authoritative for direct campaign edits. Combat-relevant changes synchronize to its linked combatant in the active encounter only; closed encounters are retained as historical snapshots.
- Notes and S.P.E.C.I.A.L. values are player-character profile information and do not affect combat calculations in this feature.
- Existing player name, character name, initiative, and active/inactive controls remain available under current campaign behavior but are not redefined by this feature.

## Constitution Check

- **Affected Layers**: Domain behavior defines character profile data and validation; application behavior owns the edit and active-encounter synchronization; persistence stores the new durable profile fields atomically; the campaign UI collects input, shows validation, and refreshes affected views.
- **Domain Rules**: Validation for S.P.E.C.I.A.L., HP, Defense, DR, and defeated-state normalization must be expressed outside the UI before UI wiring.
- **Persistence Impact**: This feature adds durable notes and S.P.E.C.I.A.L. values to player characters. Planning must name the database migration, schema and query updates, data-access regeneration, backward-compatible defaults for existing characters, and database-documentation regeneration. Existing normalized DR storage remains in use.
- **Test Impact**: Planning must include focused domain validation tests, application edit/synchronization tests, persistence migration/repository tests, and campaign UI collector/refresh tests.
- **Generation Steps**: Generated database access and database documentation must be regenerated from their source definitions; generated files must not be edited manually.
- **Layering Result**: Pass for specification. No architecture exception is requested, and all required behavior can follow the existing dependency direction.
