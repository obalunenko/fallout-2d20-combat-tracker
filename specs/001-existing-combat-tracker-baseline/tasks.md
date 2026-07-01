# Tasks: Existing Combat Tracker Baseline

**Input**: Design documents from `/specs/001-existing-combat-tracker-baseline/`
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`
**Status**: migrated baseline; tasks focus on tracing, validating, and tightening the reverse-engineered specification against existing code.

## Completion Evidence

- Setup verification updated `.gitignore` with missing Go and universal local-artifact patterns; no Docker, npm, Terraform, or Helm ignore files were required by the current repository shape.
- Traceability review is captured in `spec.md` under **Implementation Traceability** and ties FR-001 through FR-014 to domain, app, SQLite, UI, docs, and test evidence.
- User-facing encounter difficulty documentation was added to `README.md`; the baseline rules remain specified in `spec.md`.
- `.agents/` versioning and credential-safety guidance was clarified in `AGENTS.md`.
- Automated validation passed on 2026-06-20 with `go test ./internal/app -run TestBaselineSmokeFlowPersistsCampaignEncounterResourcesActionsAndLogs -count=1` and `go test ./...`.
- Startup smoke passed on 2026-06-20 by launching the app with an isolated `FALLOUT_TRACKER_DB_PATH`, applying migrations through version 42, and stopping it after 5 seconds. Full interactive Fyne click-through remains a human manual follow-up and is recorded in `quickstart.md`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another open task.
- **[Story]**: Maps to a user story from `spec.md`.
- Every task names an exact path.

## Phase 1: Setup

**Purpose**: Ensure Spec Kit context and baseline artifacts are discoverable before story-level validation.

- [X] T001 Verify active feature context in `.specify/feature.json`
- [X] T002 [P] Review project constitution constraints in `.specify/memory/constitution.md`
- [X] T003 [P] Review baseline project profile in `.specify/memory/project-profile.md`
- [X] T004 [P] Review current plan artifact list in `specs/001-existing-combat-tracker-baseline/plan.md`
- [X] T005 [P] Review requirements-quality checklist in `specs/001-existing-combat-tracker-baseline/checklists/requirements.md`

## Phase 2: Foundational Cross-Story Traceability

**Purpose**: Establish shared traceability across domain, app, store, UI, and docs before validating individual user stories.

- [X] T006 Map data entities from `specs/001-existing-combat-tracker-baseline/data-model.md` to `internal/domain/campaign.go`, `internal/domain/encounter.go`, and `internal/domain/resistance.go`
- [X] T007 Map service contracts from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/campaign_service.go`, `internal/app/encounter_service.go`, and `internal/app/combat_action_service.go`
- [X] T008 Map persistence contracts from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/store/sqlite/campaign_repository.go`, `internal/store/sqlite/encounter_repository.go`, and `internal/store/sqlite/monster_template_repository.go`
- [X] T009 Map UI workflow contracts from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/main_screen.go`, `internal/ui/fyneui/campaign_dialogs.go`, and `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T010 [P] Compare quickstart validation commands in `specs/001-existing-combat-tracker-baseline/quickstart.md` with Makefile targets in `Makefile`
- [X] T011 [P] Ensure open research questions in `specs/001-existing-combat-tracker-baseline/research.md` appear as gaps or follow-up tasks in `specs/001-existing-combat-tracker-baseline/checklists/requirements.md`

## Phase 3: User Story 1 - Manage Campaign Roster (Priority: P1)

**Goal**: Validate and tighten the migrated requirements for campaign creation, activation, roster editing, active/inactive characters, and shared resources.

**Independent Test**: The story is independently validated when campaign requirements, data model, service contract, persistence behavior, UI flow, and package tests all trace to the same behavior.

- [X] T012 [P] [US1] Trace campaign requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/domain/campaign.go`
- [X] T013 [P] [US1] Trace campaign service contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/campaign_service.go`
- [X] T014 [P] [US1] Trace campaign persistence contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/store/sqlite/campaign_repository.go`
- [X] T015 [P] [US1] Trace campaign UI contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/campaign_dialogs.go`
- [X] T016 [US1] Confirm campaign validation rules in `specs/001-existing-combat-tracker-baseline/data-model.md` match tests in `internal/app/campaign_service_test.go`
- [X] T017 [US1] Confirm inactive character behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by tests in `internal/store/sqlite/campaign_repository_test.go`
- [X] T018 [US1] Update any missing campaign requirement detail in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 4: User Story 2 - Create And Run Encounters (Priority: P1)

**Goal**: Validate and tighten migrated requirements for encounter creation, ordering, activation, editing, restart, delete, rounds, and turns.

**Independent Test**: The story is independently validated when encounter lifecycle rules and contracts trace to domain, service, repository, UI, and tests.

- [X] T019 [P] [US2] Trace encounter ordering and turn requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/domain/encounter.go`
- [X] T020 [P] [US2] Trace encounter lifecycle service contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/encounter_service.go`
- [X] T021 [P] [US2] Trace encounter persistence contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/store/sqlite/encounter_repository.go`
- [X] T022 [P] [US2] Trace encounter editor contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T023 [P] [US2] Trace encounter list/order contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/encounter_list_dialog.go`
- [X] T024 [US2] Confirm turn and defeated-combatant behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/domain/encounter_test.go`
- [X] T025 [US2] Confirm restart, activation, update, and soft-delete behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/app/encounter_service_test.go`
- [X] T026 [US2] Confirm encounter difficulty scoring requirements in `specs/001-existing-combat-tracker-baseline/spec.md` are covered by `internal/domain/encounter.go` and `internal/domain/encounter_test.go`
- [X] T027 [US2] Update any missing encounter lifecycle detail in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 5: User Story 3 - Track Combat Resources And Actions (Priority: P1)

**Goal**: Validate and tighten migrated requirements for Party AP, GM Threat, typed/location damage, healing, resistance, immunity, and operation logs.

**Independent Test**: The story is independently validated when combat action rules and contracts trace to domain, service, persistence, UI, and tests.

- [X] T028 [P] [US3] Trace resource requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/domain/encounter.go`
- [X] T029 [P] [US3] Trace damage and healing requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/domain/encounter.go`
- [X] T030 [P] [US3] Trace resistance and immunity model rules from `specs/001-existing-combat-tracker-baseline/data-model.md` to `internal/domain/resistance.go`
- [X] T031 [P] [US3] Trace combat action service contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/combat_action_service.go`
- [X] T032 [P] [US3] Trace operation log contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/operation_log.go`
- [X] T033 [P] [US3] Trace apply damage and heal UI contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/combatant_action_dialogs.go`
- [X] T034 [US3] Confirm damage, healing, AP, Threat, resistance, and immunity behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/domain/encounter_test.go`
- [X] T035 [US3] Confirm resistance profile behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/domain/resistance_test.go`
- [X] T036 [US3] Confirm non-critical audit side-effect behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/app/operation_log_test.go`
- [X] T037 [US3] Update any missing combat action or audit-log detail in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 6: User Story 4 - Reuse Monster Templates (Priority: P2)

**Goal**: Validate and tighten migrated requirements for saving, de-duplicating, upserting, listing, and loading reusable NPC templates.

**Independent Test**: The story is independently validated when monster template behavior traces through app, persistence, UI, and tests.

- [X] T038 [P] [US4] Trace monster template entity rules from `specs/001-existing-combat-tracker-baseline/data-model.md` to `internal/domain/encounter.go`
- [X] T039 [P] [US4] Trace monster template service contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/app/monster_template_service.go`
- [X] T040 [P] [US4] Trace monster template persistence contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/store/sqlite/monster_template_repository.go`
- [X] T041 [P] [US4] Trace monster template UI contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/ui/fyneui/encounter_editor_dialog.go`
- [X] T042 [US4] Confirm monster template behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/app/monster_template_service_test.go`
- [X] T043 [US4] Update any missing monster template duplicate/upsert detail in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 7: User Story 5 - Persist Local Data Safely (Priority: P2)

**Goal**: Validate and tighten migrated requirements for SQLite path resolution, migrations, normalized stats/resistance rows, generated sqlc code, and DB docs.

**Independent Test**: The story is independently validated when persistence requirements trace to DB setup, migrations, repositories, generated artifacts, docs, and tests.

- [X] T044 [P] [US5] Trace DB path and migration startup requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/store/sqlite/db.go`
- [X] T045 [P] [US5] Trace normalized stat and resistance requirements from `specs/001-existing-combat-tracker-baseline/data-model.md` to `internal/store/sqlite/normalized_stats.go`
- [X] T046 [P] [US5] Trace resistance profile persistence requirements from `specs/001-existing-combat-tracker-baseline/data-model.md` to `internal/store/sqlite/resistance_profiles.go`
- [X] T047 [P] [US5] Trace sqlc generation contract from `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md` to `internal/store/sqlite/sqlc/query.sql`
- [X] T048 [P] [US5] Trace schema documentation expectations from `specs/001-existing-combat-tracker-baseline/quickstart.md` to `docs/db/README.md`
- [X] T049 [US5] Confirm DB path behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/store/sqlite/db_test.go`
- [X] T050 [US5] Confirm schema and normalized storage behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/store/sqlite/db_schema_test.go`
- [X] T051 [US5] Update any missing persistence failure or recovery requirement in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 8: User Story 6 - Operate Through A Pip-Boy Styled Desktop UI (Priority: P3)

**Goal**: Validate and tighten migrated requirements for STAT/CAMP/DATA tabs, UI empty states, refresh behavior, action controls, and manual smoke coverage.

**Independent Test**: The story is independently validated when UI requirements trace to presenters, views, dialogs, quickstart steps, and helper tests.

- [X] T052 [P] [US6] Trace main UI state requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/ui/fyneui/main_screen.go`
- [X] T053 [P] [US6] Trace refresh behavior requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/ui/fyneui/main_view_refresher.go`
- [X] T054 [P] [US6] Trace presenter requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/ui/fyneui/main_view_presenter.go`
- [X] T055 [P] [US6] Trace active target requirements from `specs/001-existing-combat-tracker-baseline/spec.md` to `internal/ui/fyneui/active_target_view.go`
- [X] T056 [P] [US6] Trace quickstart manual smoke flow from `specs/001-existing-combat-tracker-baseline/quickstart.md` to `internal/ui/fyneui/app.go`
- [X] T057 [US6] Confirm UI presenter and refresh behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/ui/fyneui/main_view_presenter_test.go`
- [X] T058 [US6] Confirm UI collector behavior in `specs/001-existing-combat-tracker-baseline/spec.md` is covered by `internal/ui/fyneui/input_row_collectors_test.go`
- [X] T059 [US6] Update any missing Fyne UI refresh or empty-state requirement in `specs/001-existing-combat-tracker-baseline/spec.md`

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Close known brownfield gaps and keep spec artifacts internally consistent.

- [X] T060 [P] Add user-facing encounter difficulty explanation to `README.md`
- [X] T061 [P] Add or link encounter difficulty explanation in `specs/001-existing-combat-tracker-baseline/spec.md`
- [X] T062 [P] Define encounter log retention/export scope in `specs/001-existing-combat-tracker-baseline/spec.md`
- [X] T063 [P] Define Fyne accessibility and keyboard-navigation scope in `specs/001-existing-combat-tracker-baseline/spec.md`
- [X] T064 [P] Clarify `.agents/` versioning and credential safety guidance in `AGENTS.md`
- [X] T065 Run markdown consistency review for `specs/001-existing-combat-tracker-baseline/research.md`
- [X] T066 Run markdown consistency review for `specs/001-existing-combat-tracker-baseline/data-model.md`
- [X] T067 Run markdown consistency review for `specs/001-existing-combat-tracker-baseline/contracts/ui-service-contract.md`
- [X] T068 Run markdown consistency review for `specs/001-existing-combat-tracker-baseline/quickstart.md`
- [X] T069 Run `go test ./...` from repository root `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout` and record the result beside validation guidance in `specs/001-existing-combat-tracker-baseline/quickstart.md`
- [X] T070 Run the manual smoke flow from `specs/001-existing-combat-tracker-baseline/quickstart.md` and record the result in `specs/001-existing-combat-tracker-baseline/quickstart.md`

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational Traceability (Phase 2)**: Depends on Setup and blocks story validation.
- **US1, US2, US3 (P1)**: Can proceed after Phase 2 and should be completed before P2 stories when prioritizing review effort.
- **US4, US5 (P2)**: Can proceed after Phase 2; US5 should precede persistence-related follow-up specs.
- **US6 (P3)**: Can proceed after Phase 2 but benefits from US1-US3 conclusions.
- **Polish (Phase 9)**: Depends on story validation findings.

### User Story Dependencies

- **US1 Manage Campaign Roster**: Independent after Phase 2.
- **US2 Create And Run Encounters**: Depends conceptually on active campaign requirements from US1, but can be validated in parallel.
- **US3 Track Combat Resources And Actions**: Depends conceptually on active encounter requirements from US2, but can be validated in parallel.
- **US4 Reuse Monster Templates**: Extends encounter editor flows from US2.
- **US5 Persist Local Data Safely**: Cross-cuts all stories and should be reviewed before schema-changing future specs.
- **US6 Desktop UI**: Cross-cuts all stories and should incorporate findings from US1-US5.

## Parallel Execution Examples

### P1 Story Trace

```bash
# Parallelizable review tasks:
T012 internal/domain/campaign.go
T019 internal/domain/encounter.go
T028 internal/domain/encounter.go
T030 internal/domain/resistance.go
```

### Persistence Trace

```bash
# Parallelizable persistence review tasks:
T014 internal/store/sqlite/campaign_repository.go
T021 internal/store/sqlite/encounter_repository.go
T044 internal/store/sqlite/db.go
T045 internal/store/sqlite/normalized_stats.go
```

### UI Trace

```bash
# Parallelizable UI review tasks:
T015 internal/ui/fyneui/campaign_dialogs.go
T022 internal/ui/fyneui/encounter_editor_dialog.go
T053 internal/ui/fyneui/main_view_refresher.go
T055 internal/ui/fyneui/active_target_view.go
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete US1, US2, and US3 validation tasks because they cover the core live-combat loop.
3. Run `go test ./...` from repository root `/Users/olegbalunenko/Code/Go/github.com/obalunenko/fallout`.

### Incremental Review

1. Validate P1 stories and update `spec.md` with any missing requirements.
2. Validate P2 stories for reusable monsters and persistence contracts.
3. Validate P3 UI experience and manual smoke documentation.
4. Convert unresolved polish items into future feature specs when they require code changes.

### Format Validation

- All tasks use markdown checkboxes.
- All tasks use sequential IDs from T001 through T070.
- Story-phase tasks include `[US#]` labels.
- Parallel tasks use `[P]`.
- Every task includes a concrete repository path.
