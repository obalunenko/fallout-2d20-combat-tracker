# Tasks: Dynamic Encounter Difficulty Calculator

**Input**: Design documents from `/specs/002-dynamic-encounter-difficulty/`
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`
**Tests**: Required by the project constitution for domain rules, UI collection/formatting/refresh behavior, and summary regressions affected by changed metrics.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another open task.
- **[Story]**: Maps to a user story from `spec.md`.
- Every task names an exact repository path.

## Phase 1: Setup

**Purpose**: Confirm feature constraints, baseline behavior, and no-persistence scope before changing implementation.

- [X] T001 Review feature governance and active requirements in `.specify/memory/constitution.md`, `specs/002-dynamic-encounter-difficulty/spec.md`, and `specs/002-dynamic-encounter-difficulty/plan.md`
- [X] T002 [P] Review the existing ratio-based baseline in `specs/001-existing-combat-tracker-baseline/spec.md`, `README.md`, and `internal/domain/encounter.go`
- [X] T003 [P] Confirm no migration, sqlc, dbgen, or database-docs work is planned in `specs/002-dynamic-encounter-difficulty/plan.md`, `specs/002-dynamic-encounter-difficulty/data-model.md`, and `specs/002-dynamic-encounter-difficulty/contracts/ui-difficulty-preview-contract.md`

## Phase 2: Foundational Shared Difficulty Contract

**Purpose**: Establish the shared in-memory metric contract that domain, saved summaries, and UI preview will consume.

- [X] T004 Reshape `EncounterDifficulty` labels and `EncounterDifficultyMetrics` fields for Simple/Average labels, unavailable reason, XP baseline, encounter level, and difference in `internal/domain/encounter.go`
- [X] T005 Update `EncounterSummary` difficulty fields to expose the new domain-derived metrics and remove ratio/budget-only fields in `internal/domain/encounter.go`
- [X] T006 Update SQLite summary mapping to derive saved summary labels from the reshaped domain difficulty metrics in `internal/store/sqlite/mappers.go`
- [X] T007 Update summary formatting call sites to display saved summary labels from the reshaped domain difficulty metrics in `internal/ui/fyneui/formatters.go`

## Phase 3: User Story 1 - Preview Difficulty While Building An Encounter (Priority: P1)

**Goal**: The encounter editor shows a live difficulty preview from the current unsaved draft while the GM adds, removes, loads, or edits rows.

**Independent Test**: Open or construct the encounter editor with selected party rows, add/remove NPC rows, change NPC quantity/XP, and verify the preview updates from draft values without pressing Save.

- [X] T008 [P] [US1] Add draft preview collector tests for valid party rows, NPC quantity multiplying XP, valid players with no monsters, and tabletop-scale recalculation up to 12 combatants within 100ms in `internal/ui/fyneui/input_row_collectors_test.go`
- [X] T009 [P] [US1] Add editor refresh callback tests for row add, row remove, quantity, XP, level, party load, and monster load changes in `internal/ui/fyneui/encounter_editor_dialog_test.go`
- [X] T010 [US1] Replace `collectCombatantsPreviewFromRows` with a draft difficulty collector that returns player levels, monster XP times quantity, and unavailable reasons in `internal/ui/fyneui/input_rows.go`
- [X] T011 [US1] Wire `refreshDifficultyPreview` to use the draft collector and domain evaluator without save calls in `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T012 [US1] Ensure row change callbacks notify preview refresh for side, quantity/number, level, XP, and other row-level changes that may invalidate draft difficulty inputs in `internal/ui/fyneui/input_row_builders.go`
- [X] T013 [US1] Update the difficulty tab to display the current draft label and supporting metrics after party and monster library loads in `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T014 [US1] Run `go test ./internal/ui/fyneui -run 'TestCollect.*Preview|TestEncounter.*Difficulty'` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`

## Phase 4: User Story 2 - Use The Requested Difficulty Formula (Priority: P1)

**Goal**: Difficulty labels come from the requested Fallout 2d20 formula instead of the older XP-ratio model.

**Independent Test**: Evaluate known party and monster combinations that exercise every label boundary and verify metrics for average PC level, total monster XP, XP baseline, encounter level, difference, and label.

- [X] T015 [P] [US2] Replace domain difficulty tests with table cases for average PC level rounding, total monster XP, XP baseline, encounter level floor/minimum, and every label bucket in `internal/domain/encounter_test.go`
- [X] T016 [P] [US2] Add formatter tests for Unknown, Trivial, Simple, Average, Hard, Deadly, and absence of Easy/Normal wording in `internal/ui/fyneui/formatters_test.go`
- [X] T017 [US2] Implement the new formula in `EvaluateEncounterDifficulty` using total monster XP, player count, rounded-up average PC level, floored minimum encounter level, difference, and requested label buckets in `internal/domain/encounter.go`
- [X] T018 [US2] Remove old Easy/Normal ratio labels and budget-only metric behavior from the active domain difficulty evaluator in `internal/domain/encounter.go`
- [X] T019 [US2] Update `formatDifficultyPreview` and `formatEncounterDifficultySummary` to show party count, average PC level, total monster XP, XP baseline, encounter level, difference, label, Unknown state, and unavailable reason in `internal/ui/fyneui/formatters.go`
- [X] T020 [US2] Update app summary expectations for the new difficulty labels and metrics in `internal/app/encounter_service_test.go`
- [X] T021 [US2] Update SQLite mapper and repository expectations for derived new difficulty metrics in `internal/store/sqlite/mappers_test.go` and `internal/store/sqlite/encounter_repository_test.go`
- [X] T022 [US2] Update user-facing difficulty documentation to describe the new formula and labels in `README.md`
- [X] T023 [US2] Run `go test ./internal/domain ./internal/ui/fyneui ./internal/app ./internal/store/sqlite` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`

## Phase 5: User Story 3 - Keep Draft Difficulty Separate From Saves (Priority: P2)

**Goal**: Preview recalculation remains an unsaved dialog concern; stored encounters change only when the GM chooses Save.

**Independent Test**: Change monster quantities in the editor, observe preview changes, cancel the dialog, and verify saved encounter combatants remain unchanged.

- [X] T024 [P] [US3] Add UI side-effect tests proving preview changes do not invoke submit/save callbacks in `internal/ui/fyneui/encounter_editor_dialog_test.go`
- [X] T025 [P] [US3] Add service regression coverage proving saved encounter combatants stay unchanged until `UpdateEncounter` is called in `internal/app/encounter_service_test.go`
- [X] T026 [US3] Keep save-only validation and persistence collection through `collectCombatantsFromRows` inside the submit button handler in `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T027 [US3] Keep cancel behavior limited to hiding the dialog with no service calls in `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T028 [US3] Ensure invalid draft inputs produce an Unknown preview with unavailable reason without replacing save-time validation messages in `internal/ui/fyneui/input_rows.go` and `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T029 [US3] Run `go test ./internal/ui/fyneui ./internal/app` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Keep documentation, terminology, formatting, and quality gates aligned after implementation.

- [X] T030 [P] Review manual validation notes against the implemented workflow and update `specs/002-dynamic-encounter-difficulty/quickstart.md` only if needed
- [X] T031 [P] Audit active calculator-facing old terminology in `internal/domain/encounter.go`, `internal/ui/fyneui/formatters.go`, `internal/ui/fyneui/encounter_editor_dialog.go`, and `README.md`
- [X] T032 [P] Review feature requirements and planning notes against implemented scope and update `specs/002-dynamic-encounter-difficulty/spec.md` and `specs/002-dynamic-encounter-difficulty/plan.md` only if needed
- [X] T033 Run `gofmt -w` on touched Go files under `internal/domain`, `internal/ui/fyneui`, `internal/app`, and `internal/store/sqlite`
- [X] T034 Run `go test ./internal/domain ./internal/ui/fyneui` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`
- [X] T035 Run `go test ./internal/app ./internal/store/sqlite` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`
- [X] T036 Run `go test ./...` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`
- [X] T037 Run `go vet ./...` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`
- [X] T038 Run `make lint` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`
- [X] T039 Run `make build` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational Shared Difficulty Contract (Phase 2)**: Depends on Setup and blocks story implementation because domain, summary, and UI code need a shared metric shape.
- **US1 Live Preview (Phase 3)**: Depends on Phase 2. It can be implemented before or alongside US2, but acceptance should use the US2 formula.
- **US2 Formula (Phase 4)**: Depends on Phase 2. It should be completed before considering the P1 MVP done.
- **US3 Save Separation (Phase 5)**: Depends on US1 preview wiring and benefits from US2 metric correctness.
- **Polish (Phase 6)**: Depends on implemented stories and final terminology decisions.

### User Story Dependencies

- **US1 Preview Difficulty While Building An Encounter**: Independent after the shared metric contract exists; final verification should pair it with US2 formula behavior.
- **US2 Use The Requested Difficulty Formula**: Independent domain rule slice after the shared metric contract exists; UI/app/store expectations consume the same evaluator.
- **US3 Keep Draft Difficulty Separate From Saves**: Depends on US1 because it validates that the preview path is separate from the submit path.

## Parallel Execution Examples

### Setup Review

```bash
# Parallelizable review tasks:
T002 specs/001-existing-combat-tracker-baseline/spec.md README.md internal/domain/encounter.go
T003 specs/002-dynamic-encounter-difficulty/plan.md specs/002-dynamic-encounter-difficulty/data-model.md specs/002-dynamic-encounter-difficulty/contracts/ui-difficulty-preview-contract.md
```

### US1 Preview Tests

```bash
# Parallelizable before UI implementation:
T008 internal/ui/fyneui/input_row_collectors_test.go
T009 internal/ui/fyneui/encounter_editor_dialog_test.go
```

### US2 Formula Tests

```bash
# Parallelizable before formula implementation:
T015 internal/domain/encounter_test.go
T016 internal/ui/fyneui/formatters_test.go
```

### US3 Save-Separation Tests

```bash
# Parallelizable before save/cancel guard review:
T024 internal/ui/fyneui/encounter_editor_dialog_test.go
T025 internal/app/encounter_service_test.go
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete US2 formula tasks and US1 preview tasks because the live preview is only shippable when it uses the requested formula.
3. Run `go test ./internal/domain ./internal/ui/fyneui ./internal/app ./internal/store/sqlite` from `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`.

### Incremental Delivery

1. Land the domain metric contract and formula with domain tests.
2. Wire the Fyne draft collector and preview refresh with UI tests.
3. Update summary formatters and app/store expectations that consume the domain evaluator.
4. Add save/cancel separation regressions.
5. Run focused package tests, then full repository gates.

### Format Validation

- All tasks use markdown checkboxes.
- All tasks use sequential IDs from T001 through T039.
- Story-phase tasks include `[US#]` labels.
- Setup, foundational, and polish tasks do not include story labels.
- Parallel tasks use `[P]`.
- Every task includes a concrete repository path.
