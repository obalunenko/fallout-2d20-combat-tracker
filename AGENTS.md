# Agent Guidance

<!-- SPECKIT START -->
Spec Kit is installed for this repository with the Codex integration. Skills live in `.agents/skills` and are invoked as `$speckit-...`.

Recommended workflow for new work:

```text
$speckit-specify -> $speckit-clarify -> $speckit-plan -> $speckit-tasks -> $speckit-analyze -> $speckit-implement
```

For existing-code adoption and drift checks:

```text
$speckit-brownfield-scan
$speckit-brownfield-bootstrap
$speckit-brownfield-validate
$speckit-brownfield-migrate
```

Read the current feature plan and the constitution before changing code:

- `.specify/memory/constitution.md`
- `specs/001-existing-combat-tracker-baseline/spec.md`
- `specs/001-existing-combat-tracker-baseline/plan.md`
- `specs/001-existing-combat-tracker-baseline/tasks.md`
<!-- SPECKIT END -->

## Project Boundaries

- `internal/domain`: Fallout 2d20 rules and pure domain behavior. Do not import application, store, UI, or Fyne packages here.
- `internal/app`: use-case services, commands, repository contracts, operation-log side effects, and context timeouts. This layer may depend on `internal/domain`, but not on Fyne or concrete SQLite details.
- `internal/store/sqlite`: SQLite repository implementation, migrations, sqlc schema/query files, generated db access, and schema tests. Do not push SQLite-specific concerns into `internal/domain`.
- `internal/ui/fyneui`: Fyne UI composition, dialogs, presenters, collectors, refresh behavior, formatting, and UI tests. UI should call `internal/app` services instead of reimplementing rules.
- `cmd/fallout-tracker`: application assembly only.
- `docs/db`: generated database documentation; regenerate when schema changes.

## Commands

Use the narrowest command that verifies the change, then run broader gates when behavior crosses layers.

```bash
go test ./...
go vet ./...
make lint
make build
make sqlc-generate
make db-doc-generate
make goreleaser-check
```

## Spec Discipline

- New user-visible behavior starts with a spec under `specs/`.
- Brownfield specs must include `status: migrated` and cite source/test evidence.
- Persistence specs must name migration, sqlc, dbgen, and docs impact.
- UI specs must name affected Fyne screens/dialogs and refresh behavior.
- Keep checked-in Spec Kit assets versioned together: `.agents/skills`, `.specify`, and the active `specs/` feature directory should move as one set when commands regenerate them.
- Do not store secrets, tokens, connector credentials, local caches, or machine-specific state in `.agents/` or `.specify/`; this project uses those directories for Spec Kit skills, templates, extension metadata, and project guidance only.
