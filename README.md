# Fallout 2d20 Combat Tracker

Desktop combat tracker for Fallout 2d20 built entirely with Go.

## Stack

- Go
- Fyne (native desktop UI)
- SQLite (`modernc.org/sqlite`)
- Goose migrations (`github.com/pressly/goose/v3`)
- sqlc generated data-access layer
- Modular architecture (`domain` -> `app` -> `store` -> `ui`)

## Current MVP (Iteration 1)

- Encounter creation from scratch in UI
- Initiative ordering (high to low)
- Round + active turn tracking
- Next turn progression
- Party AP / GM Threat controls (`+1` / `-1`)
- Persistent local storage in SQLite
- Pip-Boy themed interface with `STAT / INV / DATA` tabs

## Create Encounter

Use `NEW ENCOUNTER` in the app header, then:

- enter encounter name
- add combatant rows with `+ Add Combatant`
- fill each row with `Name`, `Side` (`party` or `npc`), `Level`, `XP` (for `npc`), `Initiative`, `HP`, `Defense`, `DR Phys`, `DR Energy`, `DR Rad`, `DR Poison`
- in DR fields you can enter a number or `IMM` for immunity

## Storage

- DB path: `~/.config/fallout-tracker/tracker.db`
- Migrations: `internal/store/sqlite/migrations`
- Migrations are applied automatically on startup via Goose
- sqlc schema/queries:
  - `internal/store/sqlite/sqlc/schema.sql`
  - `internal/store/sqlite/sqlc/query.sql`
- generated sqlc code: `internal/store/sqlite/dbgen`
- Tooling is split into dedicated modules:
  - `tools/goose/go.tool.mod`
  - `tools/sqlc/go.tool.mod`
  - `tools/golangci-lint/go.tool.mod`

Examples:

```bash
make goose-status DB=~/.config/fallout-tracker/tracker.db
make goose-create NAME=add_new_field
make sqlc-generate
make tools-list
make tools-verify
```

## Run

```bash
go run ./cmd/fallout-tracker
```

## Verify

```bash
go test ./...
```
