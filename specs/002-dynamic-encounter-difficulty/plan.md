# Implementation Plan: Dynamic Encounter Difficulty Calculator

**Branch**: `002-dynamic-encounter-difficulty` | **Date**: 2026-06-21 | **Spec**: `specs/002-dynamic-encounter-difficulty/spec.md`
**Input**: Feature specification from `specs/002-dynamic-encounter-difficulty/spec.md`

## Summary

Add a live difficulty preview for encounter creation/editing that recalculates from the current unsaved roster draft whenever player or monster inputs change. Replace the existing ratio-based difficulty labels with the requested Fallout 2d20 formula: average selected-player level rounded up, total monster XP including quantity, encounter level from XP per player, and final label from encounter-level difference.

The rule calculation belongs in `internal/domain` so the UI, encounter summaries, and tests use one source of truth. The Fyne encounter editor will continue to collect draft rows locally and refresh the difficulty label without saving to SQLite until the GM explicitly saves the encounter.

## Technical Context

**Language/Version**: Go 1.26.3
**Primary Dependencies**: Fyne v2.7.4, modernc.org/sqlite v1.50.1, Goose v3.27.1, sqlc, google/uuid, testify
**Storage**: Local SQLite database; default path `~/.config/fallout-tracker/tracker.db`; override via `FALLOUT_TRACKER_DB_PATH`
**Testing**: `go test ./...`; focused domain tests for difficulty math; focused UI helper/formatter tests for draft preview and unavailable states; app/store tests updated only where summaries expose recalculated difficulty
**Target Platform**: Native desktop app on macOS/Linux/Windows
**Project Type**: Single Go desktop application with layered internals
**Performance Goals**: Difficulty recalculates within 100ms in focused collector/evaluator tests for tabletop-scale drafts up to 12 combatants, or passes a documented manual editor check with no visible input lag
**Constraints**: Offline-first, local-only storage, Fyne UI, no autosave for draft difficulty preview, no SQLite migration for this initial phase
**Scale/Scope**: Single GM workstation; encounter create/edit draft preview plus saved encounter summary labels derived from the same in-memory rule

## Constitution Check

*GATE: Must pass before implementation and be re-checked before completion.*

- **Layer Boundaries**: Pass. Touch `internal/domain` for the rule, `internal/ui/fyneui` for draft collection/display, and app/store tests or mapping only where existing summaries surface domain metrics. No UI imports enter domain/app/store layers.
- **Domain Rules**: Pass. Encounter difficulty evaluation changes from the older XP-ratio model to the requested encounter-level difference model in domain code before UI wiring.
- **Persistence**: Pass. No durable data shape changes, no Goose migration, no sqlc schema/query updates, and no DB docs regeneration.
- **Generated Code**: Pass. `internal/store/sqlite/dbgen` is not regenerated or manually edited.
- **Tests**: Add/update `internal/domain/encounter_test.go` for all label buckets and rounding rules; add/update `internal/ui/fyneui/*_test.go` for preview quantity, invalid draft input, and formatter text; update app/store summary tests if expected difficulty fields change.
- **Quality Gates**: Run focused package tests first (`go test ./internal/domain ./internal/ui/fyneui ./internal/app ./internal/store/sqlite` as applicable), then `go test ./...`. Run `go vet ./...` if implementation touches exported structs or cross-package contracts.

## Project Structure

### Documentation (this feature)

```text
specs/002-dynamic-encounter-difficulty/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
│   └── requirements.md
└── contracts/
    └── ui-difficulty-preview-contract.md
```

### Source Code (repository root)

```text
internal/domain/
├── encounter.go                 # difficulty labels, metrics, and evaluation rule
└── encounter_test.go            # formula, rounding, and label boundary tests

internal/app/
└── encounter_service_test.go    # summary expectations if public summary values change

internal/store/sqlite/
├── encounter_repository.go      # existing summary mapping continues to call domain rule
└── *_test.go                    # summary mapper/repository expectations if metrics change

internal/ui/fyneui/
├── encounter_editor_dialog.go   # reactive preview wiring
├── input_rows.go                # draft preview collection and validation state
├── formatters.go                # difficulty preview/summary text
└── *_test.go                    # preview collector and formatter coverage
```

**Structure Decision**: Keep the rule in `internal/domain`, keep draft UI collection/display in `internal/ui/fyneui`, and avoid persistence changes. `internal/app` and `internal/store/sqlite` are touched only if existing saved encounter summaries need expected-field adjustments after the domain metric contract changes.

## Data Model / Migration Plan

No persistence changes.

The feature introduces or reshapes in-memory difficulty result data only. No migration, backfill, sqlc query update, generated `dbgen` update, or `docs/db` regeneration is required.

## Implementation Phases

1. **Domain/Application**: Replace the ratio-based difficulty rule with the requested formula in domain code. Represent metrics needed for UI display: party count, rounded average PC level, total monster XP, XP baseline, encounter level, difference, and label. Keep missing-player or incomplete inputs from producing a misleading label. Update app-facing summary expectations to use the new label semantics.
2. **Persistence**: No schema or repository contract changes. Existing repository summary paths may continue loading combatants and deriving difficulty in memory from domain code.
3. **UI**: Update the encounter editor preview to recalculate from unsaved rows after player load, monster load, add/remove, side, level, XP, and quantity changes. Use a draft preview collector that can report invalid/incomplete inputs as unavailable instead of silently defaulting to stale values. Update preview and summary formatting to show Simple/Average terminology and the requested metric names.
4. **Docs/Generated Artifacts**: Keep this Spec Kit feature directory versioned. No database docs or generated code changes are planned.
5. **Verification**: Run focused tests for domain and UI, update any app/store tests affected by summary fields, then run `go test ./...`. Use the quickstart manual workflow to verify the Fyne editor updates without pressing Save.

## Design Artifacts

- `research.md`: decisions for rule ownership, numeric rounding, invalid draft handling, and persistence scope.
- `data-model.md`: in-memory entities and validation rules for encounter drafts and difficulty results.
- `contracts/ui-difficulty-preview-contract.md`: UI behavior contract for the encounter editor difficulty preview.
- `quickstart.md`: focused validation workflow for tests and manual UI verification.

## Post-Design Constitution Check

- **Layer Boundaries**: Pass. Design keeps Fallout difficulty rules in `internal/domain`; UI only collects draft input and displays domain results.
- **Domain Rules**: Pass. The changed mechanic is explicitly planned as domain behavior, with UI and summaries consuming it.
- **Persistence**: Pass. Planning confirms no migration, sqlc, dbgen, or DB docs work.
- **Generated Code**: Pass. No generated files are in scope.
- **Tests**: Pass. The plan names domain formula tests, UI preview tests, and summary expectation updates.
- **Quality Gates**: Pass. The planned verification starts with narrow package tests and ends with `go test ./...`; broader `go vet ./...` is included if cross-package contracts change.

## Complexity Tracking

No constitution violations are required for this feature.
