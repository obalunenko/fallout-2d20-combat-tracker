# Feature Specification: Dynamic Encounter Difficulty Calculator

**Feature Branch**: `002-dynamic-encounter-difficulty`
**Created**: 2026-06-21
**Status**: Draft
**Input**: User description: "Add a Dynamic Encounter Difficulty Calculator feature to the existing combat encounter application. The application already manages the roster of monsters and player characters. The system should automatically calculate and display the encounter's difficulty label (Simple, Average, Hard, plus Trivial and Deadly boundary labels) in real-time as monsters are added or removed from the encounter. The calculation uses average selected-player level rounded up, total monster XP, encounter level from monster XP per player, and a difficulty difference between encounter level and average PC level. For this initial phase, build this as a reactive UI component that updates immediately without requiring a backend database save each time a monster's quantity changes."

## User Scenarios & Testing

### User Story 1 - Preview Difficulty While Building An Encounter (Priority: P1)

A GM can see the current encounter difficulty label while composing or editing an encounter, so they can tune monster selection and quantity before saving changes.

**Why this priority**: The smallest valuable slice is immediate feedback during encounter setup, where the GM decides whether the current monster roster fits the selected party.

**Independent Test**: Open the encounter editor with selected party members, add or remove monster rows, change monster quantity, and verify the displayed difficulty label updates from the current draft values before saving.

**Acceptance Scenarios**:

1. **Given** an encounter draft with selected player characters, **When** the GM adds a monster with XP, **Then** the difficulty component recalculates and displays the label for the current player and monster totals.
2. **Given** an encounter draft with selected player characters and monsters, **When** the GM changes a monster quantity, **Then** the difficulty component updates immediately without requiring the GM to save the encounter.
3. **Given** an encounter draft with selected player characters and monsters, **When** the GM removes a monster, **Then** the difficulty component recalculates from the remaining monsters.

---

### User Story 2 - Use The Requested Difficulty Formula (Priority: P1)

A GM receives difficulty labels derived from the requested Fallout 2d20 encounter math rather than the older ratio-based display.

**Why this priority**: A live component is only useful if the labels match the table rule the GM expects to use during encounter balancing.

**Independent Test**: Evaluate known party and monster combinations that exercise every difficulty label boundary and verify the displayed label matches the requested formula.

**Acceptance Scenarios**:

1. **Given** selected player levels 2 and 3, **When** difficulty is calculated, **Then** the average PC level is 3 because the average is rounded up.
2. **Given** selected players and currently added monsters, **When** difficulty is calculated, **Then** total monster XP is the sum of each monster XP value multiplied by its current quantity.
3. **Given** total monster XP and player count, **When** encounter level is calculated, **Then** XP baseline is total monster XP divided by player count, encounter level is floor((XP baseline - 10) / 10), and encounter level is never below 1.
4. **Given** an encounter level and average PC level, **When** their difference is evaluated, **Then** the label is Trivial for less than -2, Simple for -2 or -1, Average for 0 or 1, Hard for 2 through 5, and Deadly for greater than 5.

---

### User Story 3 - Keep Draft Difficulty Separate From Saves (Priority: P2)

A GM can experiment with monster quantities without changing stored encounter data until they explicitly save the encounter.

**Why this priority**: The requested initial phase is a reactive preview, not autosave or persistence behavior.

**Independent Test**: Change monster quantities in the encounter editor, observe difficulty changes, cancel the dialog, and verify the saved encounter remains unchanged.

**Acceptance Scenarios**:

1. **Given** an existing saved encounter, **When** the GM opens it for editing and changes monster quantities, **Then** the displayed difficulty reflects the draft state immediately while the saved encounter remains unchanged until Save is chosen.
2. **Given** a create encounter draft, **When** the GM changes the monster roster and cancels, **Then** no new encounter is saved solely because the difficulty preview changed.

### Edge Cases

- If no player characters are selected, the calculator cannot divide monster XP by player count and must display an Unknown difficulty state with an explanatory unavailable reason until at least one player is selected.
- If players are selected but no monsters are currently added, total monster XP is 0 and the requested minimum encounter level rule still applies.
- Blank or invalid numeric draft fields must not break the encounter editor; the difficulty component must display an Unknown difficulty state with an explanatory unavailable reason until required values are valid.
- Monster quantity must affect total monster XP consistently, including rows loaded from the monster library and manually entered monster rows.
- The displayed labels for this calculator are Trivial, Simple, Average, Hard, and Deadly; older Easy or Normal terminology must not appear in this calculator.

## Requirements

### Functional Requirements

- **FR-001**: System MUST display a difficulty label for the current encounter draft whenever at least one player character is selected and all present monster draft rows contain valid XP and quantity values; a draft with selected players and no monster rows MUST calculate using Total Monster XP of 0.
- **FR-002**: System MUST calculate Average PC Level by summing selected player-character levels, dividing by selected player count, and rounding up to the nearest whole number.
- **FR-003**: System MUST calculate Total Monster XP by summing each currently added monster's XP contribution, including quantity.
- **FR-004**: System MUST calculate XP Baseline by dividing Total Monster XP by selected player count.
- **FR-005**: System MUST calculate Encounter Level by subtracting 10 from XP Baseline, dividing by 10, rounding down to the nearest whole number, and enforcing a minimum Encounter Level of 1.
- **FR-006**: System MUST calculate the difficulty difference as Encounter Level minus Average PC Level.
- **FR-007**: System MUST display Trivial when the difficulty difference is less than -2.
- **FR-008**: System MUST display Simple when the difficulty difference is -2 or -1.
- **FR-009**: System MUST display Average when the difficulty difference is 0 or 1.
- **FR-010**: System MUST display Hard when the difficulty difference is 2, 3, 4, or 5.
- **FR-011**: System MUST display Deadly when the difficulty difference is greater than 5.
- **FR-012**: System MUST update the displayed difficulty after player level, player selection, monster XP, monster side, monster quantity, monster add, monster remove, party load, or monster-library load changes in the encounter draft.
- **FR-013**: System MUST NOT require saving the encounter or writing draft quantity changes to persistent storage in order to recalculate the displayed difficulty.
- **FR-014**: System MUST preserve existing encounter save and cancel behavior; difficulty preview changes alone must not create, update, or delete encounters.
- **FR-015**: System MUST handle missing players, invalid numeric draft input, or otherwise incomplete draft data with a clear Unknown difficulty state and explanatory unavailable reason rather than a misleading label.
- **FR-016**: System MUST derive saved encounter summary difficulty labels from the same domain evaluator used by the live draft preview.

### Fallout 2d20 Rules Impact

- **Rules Affected**: Encounter difficulty evaluation and difficulty label display.
- **Rules Not Affected**: Initiative order, round advancement, Party AP, GM Threat, HP, damage, healing, resistance, immunity, monster template saving/loading, and campaign roster membership.
- **Terminology Change**: This calculator uses Trivial, Simple, Average, Hard, and Deadly. It supersedes the existing ratio-based Easy and Normal labels wherever this dynamic calculator is shown.

### Data & Persistence Impact

- **Entities/Tables**: Existing campaign player characters, encounter combatants, and monster templates provide the source values; no new entity is introduced.
- **Migration Required**: No.
- **sqlc Impact**: No schema, query, or generated database access changes are required for the initial reactive preview phase.
- **DB Docs Impact**: No database documentation regeneration is required.

### UI Impact

- **Screens/Dialogs**: Encounter create/edit dialog and its difficulty preview area; saved encounter list summaries MUST use the same domain-derived label after an encounter is saved, while live preview behavior remains the required initial interaction.
- **Input Validation**: Player count must be positive before calculation; player levels, monster XP, and monster quantity must be valid numeric draft values before a label is shown.
- **Refresh Behavior**: The difficulty component must refresh from the current unsaved encounter draft after relevant roster, level, XP, side, quantity, add, remove, party-load, and monster-load changes.

### Key Entities

- **Encounter Draft**: The in-progress encounter composition a GM is editing before save, including selected player characters and currently added monsters.
- **Selected Player Character**: A party participant whose level contributes to Average PC Level and whose count contributes to XP Baseline.
- **Monster Entry**: A non-player encounter participant whose XP and quantity contribute to Total Monster XP.
- **Difficulty Result**: The calculated Average PC Level, Total Monster XP, XP Baseline, Encounter Level, difference, and user-facing label for the current draft.

## Success Criteria

### Measurable Outcomes

- **SC-001**: In a manual encounter-editor workflow, changing monster quantity, adding a monster, or removing a monster updates the visible difficulty label within one user-observable refresh without pressing Save.
- **SC-002**: Boundary examples cover every label bucket: less than -2 displays Trivial, -2 and -1 display Simple, 0 and 1 display Average, 2 through 5 display Hard, and greater than 5 displays Deadly.
- **SC-003**: Changing draft monster quantities and canceling the editor leaves the previously saved encounter unchanged.
- **SC-004**: A tabletop-scale encounter draft with up to 12 combatants recalculates difficulty within 100ms in focused collector/evaluator tests or passes a documented manual editor check with no visible input lag.
- **SC-005**: Invalid or incomplete draft inputs never produce a divide-by-zero failure or stale misleading difficulty label.

## Assumptions

- Selected player characters are the party-side participants currently included in the encounter draft.
- Monster entries are non-party participants currently included in the encounter draft; their quantity represents repeated copies of the same monster profile for XP calculation.
- If no players are selected, Unknown is the safest displayed state because the requested formula requires division by player count.
- Persistence remains explicit: the GM saves or cancels the encounter through existing controls, and difficulty preview updates do not autosave.
- Existing local desktop and offline operation constraints continue to apply.

## Brownfield Notes

- **Status**: new.
- **Source Evidence**: `specs/001-existing-combat-tracker-baseline/spec.md`, `specs/001-existing-combat-tracker-baseline/plan.md`, `internal/domain/encounter.go`, `internal/domain/encounter_test.go`, `internal/app/encounter_service_test.go`, `internal/ui/fyneui/encounter_editor_dialog.go`, and `internal/ui/fyneui/formatters.go`.
- **Known Gaps**:
  - The existing baseline documents and implements a ratio-based difficulty model with Easy and Normal labels, so implementation planning must replace or route around that model for this calculator.
  - No automated full-window Fyne click-through test is documented for verifying live editor interaction; focused helper/presenter tests plus a concise manual workflow may be needed.
