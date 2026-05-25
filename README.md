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
- Pip-Boy themed interface with `STAT / CAMP / DATA` tabs

## Create Encounter

Use `NEW ENCOUNTER` in the app header, then:

- enter encounter name
- add combatant rows with `+ Add Combatant`
- fill each row with `Name`, `Side` (`party` or `npc`), `Level`, `XP` (for `npc`), `Initiative`, `HP`, global `Defense`, per-body `DR Phys`, `DR Energy`, `DR Rad`, and `DR Poison`
- for creatures you can enable `torso-only` to use simplified body setup (torso stats only)
- in DR fields you can enter a number or `IMM` for immunity

## Storage

- DB path: `~/.config/fallout-tracker/tracker.db` by default
- Override DB path: `FALLOUT_TRACKER_DB_PATH=/path/to/tracker.db`
- Migrations: `internal/store/sqlite/migrations`
- Migrations are applied automatically on startup via Goose
- sqlc schema/queries:
  - `internal/store/sqlite/sqlc/schema.sql`
  - `internal/store/sqlite/sqlc/query.sql`
- generated sqlc code: `internal/store/sqlite/dbgen`
- Tooling is split into dedicated modules:
  - `tools/goose/go.tool.mod`
  - `tools/sqlc/go.tool.mod`
  - `tools/tbls/go.tool.mod`
  - `tools/golangci-lint/go.tool.mod`
  - `tools/goreleaser/go.tool.mod`
- DB docs: `docs/db`

Examples:

```bash
make goose-status DB=~/.config/fallout-tracker/tracker.db
make goose-create NAME=add_new_field
make sqlc-generate
make db-doc-generate
make tools-list
make tools-verify
```

### DB Normalization Status

- Combat stats are normalized into `stat_profiles`.
- `combatants`, `player_characters`, and `monster_templates` reference a stat profile.
- Resistance rows are shared through `stat_profile_resistance_by_location`;
  global resistance uses the `body_locations.code = 'global'` dictionary row.
- Remaining deliberate snapshots are documented in `docs/db-normalization.md`.
- Legacy wide stat/resistance columns were removed by migrations.
- `internal/store/sqlite/sqlc/schema.sql` is generated from a clean database with all migrations applied.
- DB documentation is generated from the same clean migrated state via `tbls`.

## Run

```bash
make run
```

## Build (local)

```bash
make build
```

GoReleaser build for current platform only:

```bash
make goreleaser-local
```

This creates:

- `./bin/fallout-tracker-<goos>-<goarch>[.exe]`

Validate release config:

```bash
make goreleaser-check
```

## Release (tag)

Tag push `v*` triggers GitHub Actions workflow:

- file: `.github/workflows/release.yml`
- targets: `linux/amd64`, `windows/amd64`, `darwin/universal`
- behavior:
  - build Linux and Windows binaries with GoReleaser
  - build macOS universal binary with GoReleaser `universal_binaries` from `.goreleaser.darwin.yaml`
  - upload them as workflow artifacts
  - attach binaries to GitHub Release assets for that tag

Example:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Verify

```bash
go test ./...
```
