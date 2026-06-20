# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

## Summary

[Summarize the user-visible requirement and the technical approach.]

## Technical Context

**Language/Version**: Go 1.26.3
**Primary Dependencies**: Fyne v2, modernc.org/sqlite, Goose, sqlc, google/uuid, testify
**Storage**: Local SQLite database; default path `~/.config/fallout-tracker/tracker.db`; override via `FALLOUT_TRACKER_DB_PATH`
**Testing**: `go test ./...`; focused package tests for domain/app/store/ui; `go vet ./...`; `make lint`
**Target Platform**: Native desktop app on macOS/Linux/Windows
**Project Type**: Single Go desktop application with layered internals
**Performance Goals**: Local UI actions should complete without noticeable delay for tabletop-scale rosters and encounter logs
**Constraints**: Offline-first, local-only storage, Fyne UI, SQLite migrations must remain forward-only
**Scale/Scope**: Single GM workstation, multiple campaigns, multiple encounters per campaign, tabletop-sized combatant rosters

## Constitution Check

*GATE: Must pass before implementation and be re-checked before completion.*

- **Layer Boundaries**: [Which of `domain`, `app`, `store/sqlite`, `ui/fyneui`, `cmd` are touched? Confirm dependency direction remains valid.]
- **Domain Rules**: [List changed Fallout 2d20 mechanics or state "none".]
- **Persistence**: [State whether Goose migration/sqlc/schema/docs are required.]
- **Generated Code**: [State whether `internal/store/sqlite/dbgen` must be regenerated.]
- **Tests**: [List focused tests to add/update.]
- **Quality Gates**: [List commands that will be run for this feature.]

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature-name]/
├── spec.md
├── plan.md
├── tasks.md
├── research.md       # optional when design choices need exploration
├── data-model.md     # optional when persistence/domain entities change
├── quickstart.md     # optional for manual verification workflows
└── contracts/        # optional for command/service/UI contracts
```

### Source Code (repository root)

```text
cmd/fallout-tracker/
└── main.go

internal/domain/
├── encounter.go
├── resistance.go
├── campaign.go
└── *_test.go

internal/app/
├── service.go
├── *_service.go
├── repository_contracts.go
└── *_test.go

internal/store/sqlite/
├── migrations/
├── sqlc/
├── dbgen/
├── *_repository.go
└── *_test.go

internal/ui/fyneui/
├── *_dialog*.go
├── main_*.go
├── *_view*.go
└── *_test.go

docs/db/
└── generated database documentation
```

**Structure Decision**: [Name the touched directories and why the feature belongs there.]

## Data Model / Migration Plan

[If persistence changes, describe new/changed entities, migration order, backfill strategy, sqlc query changes, and docs regeneration. If none, write "No persistence changes."]

## Implementation Phases

1. **Domain/Application**: [Rules, validation, service commands, repository contract changes]
2. **Persistence**: [Migrations, sqlc schema/query/dbgen, repository changes]
3. **UI**: [Fyne dialogs/views/presenters/refresh behavior]
4. **Docs/Generated Artifacts**: [README/docs/db/spec updates]
5. **Verification**: [Focused and full checks]

## Complexity Tracking

> Fill only if the implementation violates or stretches the constitution.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g. UI directly handles rule] | [reason] | [why domain/app path was insufficient] |
