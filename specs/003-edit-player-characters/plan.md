# Implementation Plan: Edit Campaign Player Characters

**Branch**: `003-edit-player-characters` | **Date**: 2026-08-08 | **Spec**: `specs/003-edit-player-characters/spec.md`

**Input**: Feature specification from `/specs/003-edit-player-characters/spec.md`

## Summary

Extend campaign roster editing so each player row has expandable character details for notes and all seven S.P.E.C.I.A.L. attributes alongside the existing level, HP, Defense, and DR inputs. Model notes and S.P.E.C.I.A.L. as player-character-only domain data, validate them before persistence, and save them through the existing transactional campaign update.

Add migration `00043` with a notes field on player characters plus dictionary-backed S.P.E.C.I.A.L. rows seeded to 1 for existing characters. Change linked encounter reads to use stored combatant snapshots, then atomically synchronize combat-relevant values only into the most recently activated non-deleted encounter of the active campaign. This keeps the active combat view current without rewriting closed encounters.

## Technical Context

**Language/Version**: Go 1.26.3

**Primary Dependencies**: Fyne v2.7.4, modernc.org/sqlite v1.50.1, Goose v3.27.1, sqlc, google/uuid, testify

**Storage**: Local SQLite database; migration `00043` adds player-character notes and normalized S.P.E.C.I.A.L. dictionary/value rows; existing normalized stat-profile resistance storage remains unchanged

**Testing**: Focused Go tests in `internal/domain`, `internal/app`, `internal/store/sqlite`, and `internal/ui/fyneui`; schema/migration tests; full `go test ./...`, `go vet ./...`, lint, and build gates

**Target Platform**: Native desktop app on macOS, Linux, and Windows

**Project Type**: Single local desktop application with layered internals

**Performance Goals**: Validate, save, and refresh a character edit within 2 seconds for a campaign roster of up to 12 characters; focused edit and mapping operations should complete without visible UI delay

**Constraints**: Offline-first and single-user; character plus active-encounter changes must commit atomically; current HP cannot exceed maximum HP; S.P.E.C.I.A.L. values are positive integers with no feature-level upper cap; closed encounter snapshots must not change

**Scale/Scope**: One GM workstation, one active campaign, and at most one effective active encounter represented by the latest activated non-deleted encounter; campaign creation/editing and linked encounter behavior are extended without adding authentication, networking, or new combat calculations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Layer Boundaries**: Pass. `internal/domain` owns S.P.E.C.I.A.L. data and validation; `internal/app` prepares campaign edits; `internal/store/sqlite` owns transactions, migrations, mapping, and active-encounter synchronization; `internal/ui/fyneui` collects and displays values.
- **Domain Rules**: Pass. Positive S.P.E.C.I.A.L. values, HP relationships, non-negative Defense/DR, and defeated-state normalization are validated outside UI widgets.
- **Persistence**: Pass. Migration `00043` adds durable notes and normalized S.P.E.C.I.A.L. tables with backward-compatible defaults. `schema.sql`, `query.sql`, generated `dbgen`, `docs/db`, and `docs/db-normalization.md` remain aligned.
- **Generated Code**: Pass. Changes start from migrations and sqlc query sources; `internal/store/sqlite/dbgen` is regenerated, never hand-edited.
- **Tests**: Pass. The design requires domain validation, app preparation, repository atomicity/snapshot, migration/schema, UI collector/presenter, and refresh regression tests.
- **Local Desktop**: Pass. No network dependency or remote state is introduced.
- **Audit Side Effects**: Pass. Character editing introduces no required encounter audit event; existing non-critical audit behavior is unchanged.
- **Quality Gates**: Run focused package tests, generation steps, `go test ./...`, `go vet ./...`, `make lint`, and `make build`.

## Project Structure

### Documentation (this feature)

```text
specs/003-edit-player-characters/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
│   └── requirements.md
├── contracts/
│   └── player-character-edit-contract.md
└── tasks.md                         # Created later by $speckit-tasks
```

### Source Code (repository root)

```text
internal/domain/
├── campaign.go                      # player-character notes and S.P.E.C.I.A.L. value type
├── validation.go                    # reusable profile/stat validation
└── *_test.go                        # defaults, boundaries, and preservation tests

internal/app/
├── commands.go                      # campaign update payload carries character details
├── campaign_service.go              # preparation and validation before repository call
├── repository_contracts.go          # campaign repository contract remains explicit
└── campaign_service_test.go          # accepted/rejected edit behavior

internal/store/sqlite/
├── migrations/
│   └── 00043_add_player_character_details.sql
├── sqlc/
│   ├── schema.sql                   # regenerated schema snapshot
│   └── query.sql                    # notes/S.P.E.C.I.A.L. and active-encounter queries
├── dbgen/                           # regenerated sqlc output
├── campaign_repository.go           # atomic roster and active-encounter update
├── encounter_repository.go          # snapshot-based linked combatant reads/saves
├── mappers.go                       # character detail mapping
├── player_character_details.go      # normalized S.P.E.C.I.A.L. read/write helpers
├── resistance_profiles.go           # encounter reads use stored combatant DR snapshots
└── *_test.go                        # migration, mapping, atomicity, and snapshot tests

internal/ui/fyneui/
├── campaign_dialogs.go              # expandable per-character editor and refresh flow
├── input_row_builders.go            # notes and S.P.E.C.I.A.L. widgets
├── input_row_collectors.go           # parsing and field-specific validation feedback
├── input_rows.go                    # campaign row detail state
├── formatters.go                    # roster character-detail formatting as needed
└── *_test.go                        # prefill, collect, cancel, error, and refresh coverage

docs/
├── db/                              # regenerated database reference
└── db-normalization.md              # S.P.E.C.I.A.L. normalization rationale
```

**Structure Decision**: Preserve the existing domain → application → repository/UI dependency direction. Extend the current campaign editor rather than creating a second player-character workflow, store player-character-only fields outside `Combatant`, and keep encounter copies as explicit combat snapshots.

## Data Model / Migration Plan

1. Add `notes TEXT NOT NULL DEFAULT ''` to `player_characters`. Do not trim or normalize note text.
2. Add a `special_attributes` dictionary with the seven stable codes `strength`, `perception`, `endurance`, `charisma`, `intelligence`, `agility`, and `luck`.
3. Add `player_character_special_attributes`, keyed by `(player_character_id, special_attribute_id)`, with `value >= 1`, timestamps, and cascading player-character deletion.
4. Seed all seven dictionary rows and insert value `1` for every existing player character/attribute pair. New campaign characters also receive all seven values, defaulting to 1 when the create flow supplies no explicit profile.
5. Add sqlc queries to list and upsert S.P.E.C.I.A.L. rows, include notes in player-character insert/update/list paths, identify the effective active encounter only when its campaign is active, and update the matching linked combatant snapshot.
6. Make encounter reads use the combatant's stored scalar and DR profiles. Campaign data remains the source when a party member is first added; subsequent campaign edits copy combat values only to the active encounter.
7. Provide a reversible Down migration that removes S.P.E.C.I.A.L. value/dictionary storage and restores `player_characters` without notes while retaining pre-existing columns, constraints, indexes, and triggers.
8. Regenerate `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/dbgen`, and `docs/db`; update `docs/db-normalization.md`.

## Implementation Phases

1. **Domain/Application**: Introduce the player-character details and S.P.E.C.I.A.L. value object, defaults, and validation. Carry values through campaign list/create/update payloads without adding them to encounter `Combatant`. Preserve note whitespace and existing identity/initiative/immunity values.
2. **Persistence**: Add migration `00043`, sqlc queries, mappers, and normalized detail helpers. Update campaign creation/update to persist all seven attributes. In the same campaign-update transaction, synchronize level, HP, maximum HP, Defense, and DR for linked combatants in the effective active encounter only.
3. **Encounter Snapshot Semantics**: Stop substituting current player-character scalar/DR profiles when reading every linked encounter. Retain the stored combatant snapshot for closed encounters while preserving the existing copy-from-campaign behavior when a character first enters an encounter and the existing encounter-to-campaign sync when an encounter is explicitly saved.
4. **UI**: Add an expandable Character Details section to each campaign player row with multiline notes, the seven S.P.E.C.I.A.L. values, and the existing detailed DR controls. Prefill on edit, default on create, collect without trimming notes, keep entered values after validation errors, and refresh the main campaign/party/active-encounter views only after a successful outer Save.
5. **Docs/Generated Artifacts**: Regenerate schema, sqlc output, and database docs; document why S.P.E.C.I.A.L. uses dictionary-backed rows.
6. **Verification**: Run focused domain/app/UI/store tests, migration up/down and clean-schema checks, all generation steps, then full repository gates and the manual quickstart.

## Design Artifacts

- `research.md`: decisions for domain shape, normalized storage/defaults, UI integration, atomic active-encounter synchronization, and historical snapshots.
- `data-model.md`: player-character details, S.P.E.C.I.A.L. dictionary/value rows, combat profile, relationships, validation, and state transitions.
- `contracts/player-character-edit-contract.md`: campaign editor inputs, save/cancel/error behavior, refresh responsibilities, and active/closed encounter guarantees.
- `quickstart.md`: focused generation/test commands and end-to-end manual validation.

## Post-Design Constitution Check

- **Layer Boundaries**: Pass. The contract and data model assign rules, orchestration, persistence, and widgets to their existing layers.
- **Domain Rules**: Pass. All numeric and state-transition rules have domain/application test targets before UI wiring.
- **Persistence**: Pass. The design names migration `00043`, normalized S.P.E.C.I.A.L. rows, backward defaults, sqlc/schema regeneration, and DB docs updates.
- **Generated Code**: Pass. Design explicitly regenerates `dbgen` and schema/docs from sources.
- **Tests**: Pass. Design covers each affected layer plus migration, rollback, active-sync, and historical-snapshot regressions.
- **Local Desktop**: Pass. UI and storage remain local and offline.
- **Audit Side Effects**: Pass. No new critical dependency on operation logs is introduced.
- **Quality Gates**: Pass. Quickstart includes narrow tests, database generation checks, full tests, vet, lint, and build.

## Complexity Tracking

No constitution violations are required for this feature.
