# SQLite Migrations Notes

## Normalization Timeline (`00020`-`00025`)

- `00020_add_normalized_defense_tables.sql`
  - Adds `body_locations` and normalized defense tables for combatants and player characters.
  - Backfills defense values from legacy wide columns.
- `00021_add_normalized_resistance_tables.sql`
  - Adds `damage_types` and normalized resistance tables (global + per-location).
  - Backfills resistance and immunity values from legacy wide columns.
- `00022_sync_normalized_combat_stats.sql`
  - Adds sync triggers from legacy wide columns to normalized tables.
  - Transitional migration used during mixed read/write period.
- `00023_drop_legacy_combat_stats_columns.sql`
  - Drops sync triggers.
  - Removes legacy wide defense/resistance columns from:
    - `combatants`
    - `player_characters`
- `00025_drop_body_location_defense_tables.sql`
  - Drops per-body Defense tables after Defense became a global-only stat.

## Current Schema Contract

- Base entities store shared scalar fields only (`hp`, `max_hp`, `defense`, etc.).
- Defense is stored only as a global scalar on combatants and player characters.
- Per-body damage resistance is stored only in normalized resistance tables.
- `sqlc/query.sql` must not reference removed legacy columns.

## Migration Authoring Guidance

- For new combat stat dimensions, extend dictionary tables first (`body_locations`, `damage_types`) and then add row-based data, not new wide columns.
- Keep data migrations idempotent where possible (`INSERT ... ON CONFLICT` / `INSERT OR IGNORE`).
- If a migration changes write/read paths, run:
  - `go tool -modfile=tools/sqlc/go.tool.mod sqlc generate`
  - `go test ./...`
