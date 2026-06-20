# Implementation Plan: Existing Combat Tracker Baseline

**Branch**: `001-existing-combat-tracker-baseline` | **Date**: 2026-06-20 | **Spec**: `specs/001-existing-combat-tracker-baseline/spec.md`
**Input**: Reverse-engineered brownfield baseline from the existing repository.

## Summary

The current application is a local native desktop Fallout 2d20 combat tracker. It combines Fyne UI workflows, application services, domain combat rules, and SQLite persistence to manage campaigns, rosters, encounters, resources, damage/healing, monster templates, and operation logs.

## Technical Context

**Language/Version**: Go 1.26.3
**Primary Dependencies**: Fyne v2.7.4, modernc.org/sqlite v1.50.1, Goose v3.27.1, sqlc, google/uuid, testify
**Storage**: Local SQLite database with embedded Goose migrations; default path `~/.config/fallout-tracker/tracker.db`; `FALLOUT_TRACKER_DB_PATH` override
**Testing**: Go tests across `internal/domain`, `internal/app`, `internal/store/sqlite`, and `internal/ui/fyneui`
**Target Platform**: macOS/Linux/Windows desktop; OS-specific details include user config path resolution, Linux Fyne/GLFW build packages, and Windows release toolchain setup in GitHub Actions
**Project Type**: Single Go desktop app
**Performance Goals**: Responsive local tabletop workflows for up to 12 active encounter combatants and up to 100 recent encounter log entries displayed in DATA view; no strict latency SLO is defined for the migrated baseline
**Constraints**: Offline local storage, no server dependency, Fyne UI, forward migrations, sqlc generated access, no manual edits to generated `internal/store/sqlite/dbgen`
**Scale/Scope**: One GM workstation with multiple campaigns and encounters; network sync, authentication, installers/signing, and platform accessibility certification are out of scope

## Constitution Check

- **Layer Boundaries**: Pass. Current imports follow `cmd -> ui/app/store`, `ui -> app/domain`, `app -> domain`, `store/sqlite -> domain/dbgen`, and `domain` stays independent.
- **Domain Rules**: Existing mechanics are implemented in `internal/domain` and orchestrated by `internal/app`.
- **Persistence**: Existing schema uses Goose migrations, sqlc schema/query files, generated `dbgen`, and DB docs.
- **Generated Code**: `internal/store/sqlite/dbgen` is generated from sqlc and treated as generated output.
- **Tests**: Existing tests cover domain rules, services, repositories/schema, and UI helpers.
- **Quality Gates**: `go test ./...` is the baseline validation gate for this migrated spec.

## Project Structure

```text
cmd/fallout-tracker/
└── main.go                         # runtime assembly

internal/domain/
├── campaign.go                     # campaign and roster domain types
├── encounter.go                    # encounter, turn, resource, damage, healing, difficulty rules
├── resistance.go                   # resistance profile and body-location behavior
├── validation.go                   # combatant validation
└── *_test.go

internal/app/
├── campaign_service.go             # campaign use cases
├── encounter_service.go            # encounter lifecycle and resources
├── combat_action_service.go        # damage/healing use cases
├── monster_template_service.go     # monster template use cases
├── operation_log.go                # audit log side effects
├── repository_contracts.go         # persistence boundary
└── *_test.go

internal/store/sqlite/
├── db.go                           # DB path, open, migrate
├── *_repository.go                 # repository implementations
├── migrations/                     # Goose migrations
├── sqlc/                           # sqlc schema and query sources
├── dbgen/                          # generated sqlc code
└── *_test.go

internal/ui/fyneui/
├── app.go                          # Fyne app/window setup
├── main_screen.go                  # STAT/CAMP/DATA layout
├── *_dialog*.go                    # campaign, encounter, action dialogs
├── *_view*.go                      # active target and encounter order views
├── *_presenter*.go                 # display formatting and refresh presentation
└── *_test.go

docs/db/
└── generated database docs
```

**Structure Decision**: Keep the existing layered single-app structure. Future specs should extend the layer that owns the changed behavior rather than introduce new top-level app modules.

## Data Model / Migration Plan

This is a migrated baseline, so no new migration is introduced by this spec. Existing persistence includes campaigns, players, player characters, encounters, combatants, monster templates, encounter logs, dictionaries for body locations/damage types, stat profiles, and normalized resistance tables. The current migration guidance prefers dictionary/row-based stat extensions over wide columns.

## Implementation Phases

1. **Domain/Application**: Already implemented in `internal/domain` and `internal/app`.
2. **Persistence**: Already implemented in `internal/store/sqlite` with migrations, sqlc, and repository tests.
3. **UI**: Already implemented in `internal/ui/fyneui`.
4. **Docs/Generated Artifacts**: README and DB docs exist.
5. **Verification**: Run `go test ./...` after migration artifacts are added.

## Design Artifacts

- `research.md`: reverse-engineered technical decisions and alternatives.
- `data-model.md`: migrated domain/storage entities, relationships, validation, and state transitions.
- `contracts/ui-service-contract.md`: user-facing UI and service behavior contracts for the baseline workflows.
- `quickstart.md`: validation guide for exercising the baseline from tests and local app startup.

## Post-Design Constitution Check

- **Layer Boundaries**: Pass. Design artifacts preserve the existing dependency direction and assign future work to the owning layer.
- **Domain Rules**: Pass. Combat mechanics remain documented as domain/application behavior, not UI-only behavior.
- **Persistence**: Pass. Baseline documents existing migrations and generated data access; no new schema changes are introduced by planning.
- **Generated Code**: Pass. The plan documents `dbgen` as generated and does not require manual edits.
- **Tests**: Pass. Existing package test coverage is recorded; known gaps remain explicit follow-up candidates.
- **Quality Gates**: Pass for planning. Markdown-only plan artifacts require `git diff --check`; code changes still require `go test ./...`.

## Complexity Tracking

No constitution violations are required for the current baseline.

## Reverse-Engineered Technical Decisions

- **Local-only SQLite**: Chosen by current code and README; supports offline tabletop play.
- **Layered internals**: Domain rules are testable independently of Fyne and SQLite.
- **sqlc**: Repository SQL is typed and generated rather than hand-mapped through ad hoc row scanning.
- **Normalized resistances**: Body-location resistance is represented by dictionary-backed rows, while poison remains global-only.
- **Non-critical audit logs**: Logging failures are recorded but do not fail saved primary actions.
- **Fyne UI presenters/collectors**: UI data formatting and input collection are split into testable helpers.
