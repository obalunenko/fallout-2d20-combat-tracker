# Tasks: Edit Campaign Player Characters

**Input**: Design documents from `/specs/003-edit-player-characters/`
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/player-character-edit-contract.md`, `quickstart.md`
**Size**: normal (no explicit `size` recorded in `.spec-context.json`)
**Tests**: Required by the project constitution for domain validation, application orchestration, SQLite migration/repository behavior, and Fyne UI collection/refresh behavior.

## Format: `- [ ] **T###** [P?] [US#?] Description · exact/file/path`

- **[P]**: Independent within its wave because it touches a different file and has no incomplete dependency in that wave.
- **[US#]**: Maps the task to a user story from `spec.md`.
- Every implementation task names the exact repository file or generated artifact it creates or edits.

## Phase 1: Setup

**Purpose**: Confirm the active feature, governance, migration sequence, and current campaign/encounter behavior before implementation.

**Wave 1 — implementation baseline:**

- [X] **T001** Review the active specification, plan, constitution gates, current migration `00042`, campaign editor, campaign repository, and linked encounter read behavior before editing · `specs/003-edit-player-characters/spec.md`, `specs/003-edit-player-characters/plan.md`, `.specify/memory/constitution.md`, `internal/store/sqlite/migrations/00042_move_resources_to_campaigns.sql`, `internal/ui/fyneui/campaign_dialogs.go`, `internal/store/sqlite/campaign_repository.go`, `internal/store/sqlite/sqlc/query.sql`

## Phase 2: Foundational Character Details And Persistence

**Purpose**: Establish shared domain data, migration, typed queries, and normalized mapping that block all three user stories.

### Tests

**Wave 1 — independent failing foundation tests (different files):**

- [X] **T002** [P] Add failing tests for default and complete seven-field S.P.E.C.I.A.L. profiles, positive-value validation, and exact notes preservation · `internal/domain/campaign_test.go`
- [X] **T003** [P] Add failing schema/migration tests for `00043` Up/Down, notes defaults, seven dictionary rows, seven value-1 rows per existing character, constraints, foreign keys, indexes, and retained pre-existing player-character data · `internal/store/sqlite/db_schema_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent domain and migration sources (different files):**

- [X] **T004** [P] Add the player-character notes field, S.P.E.C.I.A.L. value object, seven stable attribute codes, default constructor, completeness checks, and positive-value validation without adding character-only fields to `Combatant` · `internal/domain/campaign.go`
- [X] **T005** [P] Create reversible migration `00043` with `player_characters.notes`, the `special_attributes` dictionary, normalized `player_character_special_attributes`, stable seeds, legacy backfill to 1, checks, timestamps, cascading deletion, and a Down path preserving the prior schema · `internal/store/sqlite/migrations/00043_add_player_character_details.sql`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — query source:**

- [X] **T006** Add sqlc source queries for notes-aware player-character insert/update/list operations plus S.P.E.C.I.A.L. dictionary listing, character-value listing, and idempotent value upsert · `internal/store/sqlite/sqlc/query.sql`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — generated persistence contract:**

- [X] **T007** Run `make sqlc-generate` to regenerate the clean migrated schema and typed database access from migration/query sources; do not hand-edit generated output · `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/dbgen/`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — independent mapping helpers (different files):**

- [X] **T008** [P] Implement normalized S.P.E.C.I.A.L. dictionary validation, complete-profile assembly, default insertion, and seven-value upsert helpers with clear inconsistent-data errors · `internal/store/sqlite/player_character_details.go`
- [X] **T009** [P] Extend player-character SQL parameter and row mapping to carry exact notes and attach assembled S.P.E.C.I.A.L. details while preserving identity, initiative, availability, and existing combat profile fields · `internal/store/sqlite/mappers.go`

**Checkpoint**: Domain and typed persistence foundations compile conceptually; no user story implementation starts until T001–T009 are complete.

## Phase 3: User Story 1 — Edit A Character Profile (Priority: P1)

**Goal**: A GM can open the existing campaign editor, expand a character’s details, edit notes/S.P.E.C.I.A.L./combat values/DR, save atomically, and reopen the same stored values.

**Independent Test**: Edit every in-scope field for one campaign character, save, reload the campaign, and verify an exact round trip; then cancel a second draft and verify no persistence.

### Tests

**Wave 1 — independent failing story tests (different files):**

- [X] **T010** [P] [US1] Add application tests for default details on campaign creation, exact multiline-note preservation, complete S.P.E.C.I.A.L. propagation, and preservation of existing identity/initiative/immunity fields · `internal/app/campaign_service_test.go`
- [X] **T011** [P] [US1] Add repository tests for create/list/update/reopen round trips of notes, seven S.P.E.C.I.A.L. values, level, HP, Defense, DR, immunity, availability, and transactional rollback on injected write failure · `internal/store/sqlite/campaign_repository_test.go`
- [X] **T012** [P] [US1] Add UI collector tests for blank/exact multiline notes, seven S.P.E.C.I.A.L. fields, existing combat fields, collapsed detail collection, DR/immunity preservation, and create defaults · `internal/ui/fyneui/input_row_collectors_test.go`
- [X] **T013** [P] [US1] Add campaign dialog tests for existing-value prefill, expand/collapse behavior, successful Save callback payload/refresh, and Cancel without a persistence callback · `internal/ui/fyneui/campaign_dialogs_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — independent application, persistence, and widget construction (different files):**

- [X] **T014** [P] [US1] Extend campaign create/update preparation to supply defaults for new characters, validate complete details, preserve exact notes, and pass all profile fields to the repository without changing encounter `Combatant` · `internal/app/campaign_service.go`
- [X] **T015** [P] [US1] Persist and load player-character notes and all seven S.P.E.C.I.A.L. values inside existing campaign transactions for create, rename/version, same-character update, active/inactive, and list flows · `internal/store/sqlite/campaign_repository.go`
- [X] **T016** [P] [US1] Add campaign-row state and expandable Character Details widgets for multiline notes and seven S.P.E.C.I.A.L. entries while retaining the compact base row and existing DR controls · `internal/ui/fyneui/input_rows.go`, `internal/ui/fyneui/input_row_builders.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — UI collection:**

- [X] **T017** [US1] Collect exact notes, S.P.E.C.I.A.L., existing combat values, DR, immunity, and hidden collapsed values into campaign player payloads with stable create defaults · `internal/ui/fyneui/input_row_collectors.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — campaign editor integration:**

- [X] **T018** [US1] Prefill expanded details for existing characters, reset defaults for new/cleared rows, submit the complete outer campaign edit once, close only on success, invoke full refresh once, and keep Cancel side-effect-free · `internal/ui/fyneui/campaign_dialogs.go`

**Checkpoint**: User Story 1 is independently functional and testable through the campaign editor and persistence round trip.

## Phase 4: User Story 2 — Prevent Invalid Character Stats (Priority: P1)

**Goal**: Invalid level, S.P.E.C.I.A.L., HP, Defense, or DR values produce field-specific feedback, retain the draft, and mutate no stored state.

**Independent Test**: Submit every invalid boundary and a multiple-error draft; verify feedback, retained input, zero repository mutation, and unchanged stored values.

### Tests

**Wave 1 — independent failing validation tests (different files):**

- [X] **T019** [P] [US2] Expand domain tests for zero/negative S.P.E.C.I.A.L., level below 1, HP boundaries, Defense/DR negativity, exact notes behavior, and no upper cap on positive S.P.E.C.I.A.L. values · `internal/domain/campaign_test.go`
- [X] **T020** [P] [US2] Add application tests proving invalid player-character details fail before repository mutation and identify the affected player/character field · `internal/app/campaign_service_test.go`
- [X] **T021** [P] [US2] Add UI collector tests for invalid/non-integer S.P.E.C.I.A.L., HP relationships, negative Defense/DR, multiple invalid inputs, and field-specific messages · `internal/ui/fyneui/input_row_collectors_test.go`
- [X] **T022** [P] [US2] Add dialog tests proving validation/persistence errors keep the editor open, preserve entered values, suppress refresh, and leave Cancel side-effect-free after an error · `internal/ui/fyneui/campaign_dialogs_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — domain validation source:**

- [X] **T023** [US2] Finalize reusable player-character detail validation so all seven S.P.E.C.I.A.L. fields, level, HP, Defense, and the existing resistance profile produce stable field-specific errors before persistence · `internal/domain/campaign.go`, `internal/domain/validation.go`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — independent application and UI enforcement (different files):**

- [X] **T024** [P] [US2] Apply domain detail validation in campaign preparation without trimming notes or normalizing invalid HP into an accepted value, and return player/character-qualified errors · `internal/app/campaign_service.go`
- [X] **T025** [P] [US2] Parse and validate every campaign detail field while retaining widget text, report precise labels, and preserve existing resistance/immunity validation behavior · `internal/ui/fyneui/input_row_collectors.go`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — error-state UI integration:**

- [X] **T026** [US2] Keep the campaign editor visible with all draft values after validation or save errors, show actionable feedback, avoid refresh/close on failure, and preserve Cancel semantics · `internal/ui/fyneui/campaign_dialogs.go`

**Checkpoint**: User Story 2 is independently functional and testable; every invalid boundary is rejected without stored mutation or lost draft input.

## Phase 5: User Story 3 — Keep Active Combat In Sync (Priority: P2)

**Goal**: A campaign edit atomically updates a linked combatant in the effective active encounter while closed encounters retain stored snapshots.

**Independent Test**: Put one character in two encounters, activate one, edit the campaign character, and verify only the active encounter matches the new combat values/defeated state while round, turn, unrelated combatants, notes/S.P.E.C.I.A.L. boundaries, and the closed encounter remain unchanged.

### Tests

**Wave 1 — independent failing synchronization tests (different files):**

- [X] **T027** [P] [US3] Add campaign repository tests for active-campaign detection, effective active-encounter selection, atomic linked scalar/DR/immunity sync, defeated-state derivation, preserved round/turn/position/unrelated combatants, no sync for inactive campaigns, and rollback · `internal/store/sqlite/campaign_repository_test.go`
- [X] **T028** [P] [US3] Add encounter repository regressions proving linked combatants read their stored scalar/DR snapshot, new party additions still copy campaign values, explicit encounter saves still sync linked combat state back, and closed encounters remain unchanged after campaign edits · `internal/store/sqlite/encounter_repository_test.go`
- [X] **T029** [P] [US3] Add UI refresh tests proving successful campaign edits reload campaign roster, party library, active encounter order/target/tactical state, and defeated indicators while failures leave the rendered state intact · `internal/ui/fyneui/main_view_refresher_test.go`

### Implementation

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — SQL snapshot and active-selection contract:**

- [X] **T030** [US3] Change linked encounter reads to stored combatant scalar/DR profiles and add queries that identify the effective active encounter only for the active campaign plus its linked combatant snapshots · `internal/store/sqlite/sqlc/query.sql`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — regenerate typed queries:**

- [X] **T031** [US3] Run `make sqlc-generate` after snapshot/active-selection query changes and verify generated query types match the migration and mapper contract · `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/dbgen/`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — independent repository implementations (different files):**

- [X] **T032** [P] [US3] Stop overriding encounter combatant resistance with the current player-character profile so every encounter loads its stored DR/immunity snapshot · `internal/store/sqlite/resistance_profiles.go`
- [X] **T033** [P] [US3] Synchronize level, current/max HP, Defense, DR/immunity, and defeated state into linked combatant stat profiles in the effective active encounter inside the existing campaign transaction while preserving all unrelated encounter state · `internal/store/sqlite/campaign_repository.go`, `internal/store/sqlite/normalized_stats.go`
- [X] **T034** [P] [US3] Preserve copy-from-campaign behavior for newly linked party members and encounter-to-campaign synchronization for explicitly saved encounters under the new snapshot read semantics · `internal/store/sqlite/encounter_repository.go`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — successful-save refresh integration:**

- [X] **T035** [US3] Ensure the campaign Save success path invokes the existing full main refresh so the active encounter order, target, tactical summary, party library, roster, and defeated state immediately reflect synchronized values · `internal/ui/fyneui/campaign_dialogs.go`, `internal/ui/fyneui/main_view_refresher.go`

**Checkpoint**: User Story 3 is independently functional and testable; active combat matches the campaign edit and closed encounters remain historical snapshots.

## Phase 6: Polish And Cross-Cutting Validation

**Purpose**: Align generated artifacts/documentation, format code, validate all success criteria, and record manual evidence.

**Wave 1 — independent documentation updates (different files):**

- [X] **T036** [P] Document the S.P.E.C.I.A.L. dictionary/value normalization, notes ownership, migration defaults, and encounter snapshot boundary · `docs/db-normalization.md`
- [X] **T037** [P] Review the implemented behavior against the feature contract and refine manual/automated validation steps or expected outcomes where implementation details require clarification · `specs/003-edit-player-characters/quickstart.md`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — source formatting:**

- [X] **T038** Run `gofmt -w` on all touched Go source and test files under the affected layers · `internal/domain/`, `internal/app/`, `internal/store/sqlite/`, `internal/ui/fyneui/`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — final generated artifact verification:**

- [X] **T039** Run `make sqlc-generate` and `make db-doc-generate`; verify clean-schema sqlc output and regenerated database docs match migration `00043` without manual generated-code edits · `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/dbgen/`, `docs/db/`

**⟶ Wait for Wave 3 to finish, then:**

**Wave 4 — focused layer validation:**

- [X] **T040** Run `go test ./internal/domain ./internal/app ./internal/store/sqlite ./internal/ui/fyneui` and fix any feature regressions before broader gates · `internal/domain/`, `internal/app/`, `internal/store/sqlite/`, `internal/ui/fyneui/`

**⟶ Wait for Wave 4 to finish, then:**

**Wave 5 — single-owner Success Criteria suite:**

- [X] **T041** Validate SC-002, SC-003, SC-004, and SC-006 with `go test ./...`, then run `go vet ./...`, `make lint`, and `make build`; record command results · `specs/003-edit-player-characters/quickstart.md`

**⟶ Wait for Wave 5 to finish, then:**

**Wave 6 — manual acceptance:**

- [X] **T042** Execute the isolated-database manual workflow for SC-001 and SC-005, record observed timing/usability plus restart and closed-encounter results, and explicitly document any human follow-up that cannot be completed automatically · `specs/003-edit-player-characters/quickstart.md`

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup** starts immediately and establishes the reviewed baseline.
- **Phase 2 Foundational** depends on Setup and blocks every user story; its waves run tests → domain/migration → queries → generation → mapping helpers.
- **Phase 3 US1** depends on Foundational and delivers the editable/persisted character profile MVP; its waves run tests → independent layer implementation → collection → dialog integration.
- **Phase 4 US2** depends on US1’s editor/persistence path and hardens it; its waves run tests → domain validation → application/UI enforcement → error-state integration.
- **Phase 5 US3** depends on US1 persistence and US2 validation; its waves run tests → SQL contract → generation → repository changes → UI refresh integration.
- **Phase 6 Polish** depends on all stories; its waves run docs → formatting → generation/docs → focused tests → full Success Criteria gates → manual acceptance.

### User Story Delivery Order

1. **US1 Edit A Character Profile** is the MVP and becomes testable after T018.
2. **US2 Prevent Invalid Character Stats** builds on US1 and becomes testable after T026.
3. **US3 Keep Active Combat In Sync** builds on the valid atomic edit path and becomes testable after T035.
4. **Polish** validates the integrated feature and completion criteria after all story checkpoints.

### Wave Restatement

- **Setup**: T001.
- **Foundational**: T002–T003 → T004–T005 → T006 → T007 → T008–T009.
- **US1**: T010–T013 → T014–T016 → T017 → T018.
- **US2**: T019–T022 → T023 → T024–T025 → T026.
- **US3**: T027–T029 → T030 → T031 → T032–T034 → T035.
- **Polish**: T036–T037 → T038 → T039 → T040 → T041 → T042.
