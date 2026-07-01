# Feature Specification: Existing Combat Tracker Baseline

**Feature Branch**: `001-existing-combat-tracker-baseline`
**Created**: 2026-06-20
**Status**: migrated
**Input**: Reverse-engineered from the current repository with the Spec Kit brownfield workflow.

## User Scenarios & Testing

### User Story 1 - Manage Campaign Roster (Priority: P1)

A GM can create, open, and edit campaigns with players and active characters so encounters are tied to a party roster instead of anonymous combatants.

**Why this priority**: Campaigns are the root context for encounters, party resources, and party character reuse.

**Independent Test**: Create a campaign with at least one player/character, list campaigns, activate the campaign, edit roster entries, and verify inactive/renamed characters are handled consistently.

**Acceptance Scenarios**:

1. **Given** no active campaign, **When** a GM creates a campaign with name, start date, and players, **Then** the campaign is persisted and becomes active when no other active campaign exists.
2. **Given** an existing campaign, **When** a GM edits player names or character stats, **Then** active player characters are updated or versioned according to name changes.
3. **Given** a player is removed or made inactive, **When** campaign data is saved, **Then** inactive party characters are removed from existing encounters.

### User Story 2 - Create And Run Encounters (Priority: P1)

A GM can create an encounter from manual combatants, saved party members, or monster templates, then run initiative order round by round.

**Why this priority**: Encounter creation and turn progression are the core combat-tracking loop.

**Independent Test**: Create an encounter, verify initiative sorting, activate it, advance turns, and confirm defeated combatants are skipped while rounds increment.

**Acceptance Scenarios**:

1. **Given** an active campaign, **When** a GM creates an encounter with combatants, **Then** combatants are sorted by initiative descending with name tie-breaks and the first combatant is active.
2. **Given** an encounter with defeated combatants, **When** the GM advances turns, **Then** defeated combatants are skipped and the round increments after wrapping the order.
3. **Given** a saved encounter, **When** the GM launches, edits, restarts, or soft-deletes it, **Then** list state and active encounter state reflect that operation.

### User Story 3 - Track Combat Resources And Actions (Priority: P1)

A GM can adjust party AP and GM Threat, apply typed/location damage, and heal combatants while HP, defeated state, and party character state persist.

**Why this priority**: These are the repeated in-combat actions the tracker exists to reduce.

**Independent Test**: Apply resource changes, damage, and healing through services and verify persisted encounter state plus operation logs.

**Acceptance Scenarios**:

1. **Given** an active encounter, **When** party AP is added beyond 6, **Then** party AP is capped at 6.
2. **Given** insufficient AP or Threat, **When** the GM spends more than available, **Then** the operation fails and state is not reduced below zero.
3. **Given** a combatant with resistance or immunity, **When** typed damage is applied to a body location, **Then** effective damage respects resistance, immunity, and torso-only behavior.
4. **Given** a defeated combatant, **When** healing restores HP above zero, **Then** the combatant is no longer defeated.

### User Story 4 - Reuse Monster Templates (Priority: P2)

A GM can save NPC combatants as reusable monster templates and later load them into encounter rows.

**Why this priority**: Reuse reduces repeated manual data entry after the core combat loop works.

**Independent Test**: Save NPC combatants, list templates, load one with a chosen count, and verify party combatants are excluded from monster saves.

**Acceptance Scenarios**:

1. **Given** an encounter editor containing NPC rows, **When** the GM saves NPCs to the database, **Then** unique monster templates are upserted by name.
2. **Given** saved monster templates, **When** the GM loads a template into an encounter, **Then** it is added as an NPC row with generated encounter-specific IDs.

### User Story 5 - Persist Local Data Safely (Priority: P2)

The tracker persists campaigns, encounters, combatants, normalized stats/resistances, resources, and logs in a local SQLite database that migrates on startup.

**Why this priority**: The application must survive restarts and schema evolution.

**Independent Test**: Open/migrate a database, save/read campaigns and encounters, and run schema/repository tests.

**Acceptance Scenarios**:

1. **Given** no `FALLOUT_TRACKER_DB_PATH`, **When** the app starts, **Then** it uses the default config database path.
2. **Given** `FALLOUT_TRACKER_DB_PATH`, **When** the app starts, **Then** it opens that SQLite database and creates parent directories when needed.
3. **Given** a migrated database, **When** combat stats are saved, **Then** scalar stats and normalized resistance rows remain consistent.

### User Story 6 - Operate Through A Pip-Boy Styled Desktop UI (Priority: P3)

A GM can use a native Fyne UI with STAT, CAMP, and DATA tabs, responsive dialogs, tactical summaries, and action controls.

**Why this priority**: The UI makes the domain behavior usable during live play.

**Independent Test**: Exercise UI collectors, presenters, formatters, and controller tests; manually run the app when UI layout changes.

**Acceptance Scenarios**:

1. **Given** no active campaign, **When** the app opens, **Then** the UI prompts the GM to create or open a campaign.
2. **Given** an active campaign but no active encounter, **When** the UI refreshes, **Then** it prompts the GM to create or open an encounter.
3. **Given** an active encounter, **When** state-changing actions complete, **Then** STAT/CAMP/DATA displays refresh from service state.

## Edge Cases

- Empty encounter creation is rejected.
- Campaign creation and update require campaign name, start date, and at least one player.
- Combatants require valid side, non-negative level/XP/initiative/HP/max HP/defense, and non-negative resistance.
- Current HP cannot exceed max HP.
- Poison resistance is global-only; physical, energy, and radiation resistance are location-based.
- Torso-only combatants resolve location damage through torso resistance.
- Duplicate linked party characters in one encounter are rejected.
- No valid next combatant exists when all combatants are defeated.
- Operation log write failures are recorded as non-critical side effects and do not fail the primary action after state save.

## Requirement Clarifications

### Campaign Resources And Party Character Sync

- Party AP and GM Threat are campaign-level resources. Encounters display and mutate the active campaign's resources rather than owning independent resource counters.
- Encounter creation copies the active campaign resources into displayed encounter state, and resource updates are persisted through the campaign resource path.
- Linked party combatants preserve their `PlayerCharacterID`. HP, defeated state, active state, and normalized combat profile changes for linked party combatants are allowed to synchronize back to active campaign characters when saved.
- Unlinked NPC combatants and monster-template-derived combatants never synchronize into campaign player characters.

### Monster Template Identity

- Monster template names are trimmed before validation and persistence.
- Saving multiple NPCs with the same lower-cased name in one save request keeps the first prepared template and ignores later duplicates in that batch.
- Persisting a monster with a name that already exists updates the existing template instead of creating a second template row.
- Loaded monster templates must enter encounters as fresh NPC combatants with encounter-specific IDs and no player-character link.

### Encounter Difficulty

- The user-facing difficulty summary is documented in `README.md`; this section is the baseline rules source for future implementation changes.
- Difficulty is derived only when both party and enemy combatants are present; otherwise the difficulty label is `Unknown`.
- Party XP budget is `(average party level + 1) * party combatant count * 10`, rounded to the nearest integer for display.
- Difficulty score is `enemy total XP / party XP budget`, rounded to two decimal places for display.
- Difficulty labels are selected by score: `< 0.5` = `Trivial`, `< 1.0` = `Easy`, `< 1.5` = `Normal`, `<= 2.25` = `Hard`, and `> 2.25` = `Deadly`.

### Exception And Recovery Flows

- Missing active campaign prevents campaign-scoped encounter operations and should surface a recoverable application error rather than silently creating global data.
- Missing active encounter prevents turn, resource, damage, heal, and log-read workflows that require encounter state.
- Blank campaign IDs, encounter IDs, combatant IDs, and log messages are invalid where those IDs/messages are required.
- Repository not-found cases for campaigns or encounters must surface domain not-found errors.
- Database open, path resolution, or migration failure prevents application startup and must not create partially initialized runtime state.
- Transactional repository failures must roll back the attempted write and return an error to the service/UI boundary.
- Audit-log append failures are the only documented non-critical write failure in the migrated baseline; they must be logged and counted without failing the already-saved primary action.

### UI Refresh Requirements

- Campaign create, edit, and activate workflows refresh campaign overview, active roster, party library, and no-campaign/no-encounter empty states.
- Encounter create, edit, launch, restart, delete, and turn advancement refresh active encounter state, encounter order, active target, tactical snapshot, and DATA log where available.
- Party AP, GM Threat, damage, and heal workflows refresh resource labels, active target details, encounter order status, tactical snapshot, and DATA log where available.
- UI refresh requirements apply to the active in-memory Fyne screen after service operations complete successfully; failed operations must leave the previously displayed state intact except for visible validation/error feedback.

### Non-Functional Boundaries

- The migrated baseline targets tabletop-scale use: up to 12 active combatants in an encounter and up to 100 recent encounter log entries displayed in the DATA view without noticeable local UI delay.
- Fyne dialogs are considered responsive for the migrated baseline when roster/combatant tables remain scrollable and usable at the default application window size of 1100x700; mobile or small-screen layout behavior is out of scope.
- The migrated baseline supports macOS, Linux, and Windows desktop builds through Go/Fyne and the release workflow; OS-specific packaging, signing, installer behavior, and platform accessibility certification are out of scope.
- OS compatibility depends on OS-specific user config directory resolution, SQLite file paths, Linux Fyne/GLFW build dependencies, and Windows release build toolchain setup described in CI/release workflows.
- Keyboard navigation and accessibility requirements are not specified for the migrated baseline beyond default Fyne widget behavior; they require a future dedicated UI accessibility spec.
- Encounter log retention, export, compaction, and cleanup are intentionally out of scope for the migrated baseline; logs are append-only until a future log-management spec defines otherwise.

## Requirements

### Functional Requirements

- **FR-001**: System MUST manage campaigns with name, start date, players, active/inactive player characters, and shared campaign resources.
- **FR-002**: System MUST create, update, activate, restart, list, and soft-delete encounters scoped to the active campaign.
- **FR-003**: System MUST sort encounter combatants by initiative descending, tie-breaking by name, and track active turn and round.
- **FR-004**: System MUST skip defeated combatants when advancing turns and reject turn advancement when all combatants are defeated.
- **FR-005**: System MUST track Party AP with a maximum of 6 and GM Threat with a minimum of 0.
- **FR-006**: System MUST apply typed damage and healing while preserving max HP boundaries and defeated state.
- **FR-007**: System MUST support physical, energy, radiation, and poison damage types, with location-specific resistance for physical/energy/radiation and global poison resistance.
- **FR-008**: System MUST support immunity flags that reduce effective damage to zero for the affected damage type.
- **FR-009**: System MUST evaluate encounter difficulty from party count/average level and enemy XP totals.
- **FR-010**: System MUST persist local data in SQLite and run embedded Goose migrations on startup.
- **FR-011**: System MUST store normalized combat stat/resistance data for combatants, player characters, stat profiles, and monster templates.
- **FR-012**: System MUST save and load monster templates for NPC reuse.
- **FR-013**: System MUST append encounter operation logs when encounters are created, activated, updated, or restarted; turns advance; Party AP changes; GM Threat changes; damage is applied; or healing is applied, when log storage is available.
- **FR-014**: System MUST expose the implemented workflows through the Fyne desktop UI.

### Key Entities

- **Campaign**: Active play context with name, start date, resources, players, and player characters.
- **Player / PlayerCharacter**: Roster identity and active/inactive party combatant profile.
- **Encounter**: Campaign-scoped combat state with round, turn index, combatants, and resources.
- **Combatant**: Party or NPC participant with stats, HP, defense, body-location resistance, immunity, active state, and defeated state.
- **MonsterTemplate**: Reusable NPC combatant profile stored independently from encounters.
- **StatProfile / Resistance Rows**: Normalized storage contract for stats and damage resistance.
- **EncounterLog**: Round-stamped audit message shown in the DATA view.

## Success Criteria

- **SC-001**: `go test ./...` passes for the migrated baseline.
- **SC-002**: Campaign, encounter, combat action, repository, and UI helper tests cover the existing core workflows.
- **SC-003**: A GM can start the app, create a campaign, create an encounter, advance turns, change resources, apply damage/healing, and reopen persisted state.
- **SC-004**: Database schema documentation and sqlc generated code reflect all applied migrations.
- **SC-005**: Every functional requirement FR-001 through FR-014 has at least one traceability task, contract section, or validation note in the migrated Spec Kit artifacts.

## Assumptions

- The app is single-user and local-first, which constrains the baseline away from authentication, authorization, multi-user conflict resolution, and network sync requirements.
- SQLite is the system of record, so persistence, migration, and generated sqlc/schema/docs requirements apply to all durable campaign, encounter, roster, template, and log data.
- Networked play, authentication, and multi-device sync are out of scope for the migrated baseline and require new feature specs if introduced.
- Manual visual verification is still useful for Fyne layout changes even when package tests pass, because the migrated baseline has no full-window automated UI click-through test.
- Corrections to already-documented migrated behavior may update this baseline spec; new user-visible capabilities should start as separate future specs.

## Brownfield Notes

- **Status**: migrated.
- **Source Evidence**: `README.md`, `Makefile`, `go.mod`, `.github/workflows/*.yml`, `internal/domain`, `internal/app`, `internal/store/sqlite`, `internal/ui/fyneui`, `docs/db`, and test names/coverage across those packages.
- **Known Gaps**:
  - No automated end-to-end UI test launches the Fyne app and exercises a full click workflow.
  - Encounter logs are append-only; there is no retention/compaction policy.
  - Difficulty scoring now has a user-facing `README.md` summary, but future gameplay tuning should still happen through a dedicated feature spec.
  - `.agents/` is intentionally present for Spec Kit skills; contributors must avoid putting secrets there.

## Implementation Traceability

- **Campaign roster**: `internal/domain/campaign.go`, `internal/app/campaign_service.go`, `internal/store/sqlite/campaign_repository.go`, `internal/ui/fyneui/campaign_dialogs.go`, and campaign service/repository tests cover campaign creation, activation, roster update, inactive characters, and campaign resources.
- **Encounter lifecycle**: `internal/domain/encounter.go`, `internal/app/encounter_service.go`, `internal/store/sqlite/encounter_repository.go`, encounter editor/list UI files, and domain/app tests cover ordering, activation, restart, soft delete, defeated skips, rounds, and difficulty metrics.
- **Combat actions and logs**: `internal/domain/encounter.go`, `internal/domain/resistance.go`, `internal/app/combat_action_service.go`, `internal/app/operation_log.go`, action dialogs, and operation-log tests cover resource changes, damage, healing, resistance, immunity, and non-critical audit log behavior.
- **Monster templates**: `internal/app/monster_template_service.go`, `internal/store/sqlite/monster_template_repository.go`, `internal/ui/fyneui/encounter_editor_dialog.go`, and monster template tests cover trimming, duplicate suppression, upsert by normalized name, and loading templates as fresh NPC rows.
- **Persistence**: `internal/store/sqlite/db.go`, migrations, `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/sqlc/query.sql`, generated `dbgen`, `docs/db`, and SQLite schema/repository tests cover DB path resolution, migrations, normalized stats/resistance rows, generated query contracts, and schema documentation.
- **Desktop UI**: `internal/ui/fyneui/app.go`, `main_screen.go`, `main_view_refresher.go`, `main_view_presenter.go`, `active_target_view.go`, and UI helper/controller tests cover STAT/CAMP/DATA views, empty states, refresh behavior, action controls, and the startup smoke described in `quickstart.md`.
