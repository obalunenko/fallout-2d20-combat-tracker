# Quickstart: Dynamic Encounter Difficulty Calculator

## Prerequisites

- Repository root: `fallout`
- Go toolchain matching the project module
- Existing local development prerequisites for Fyne tests

## Focused Automated Validation

Run the narrow tests first while implementing:

```bash
go test ./internal/domain ./internal/ui/fyneui
```

If saved encounter summaries are updated to expose renamed or changed difficulty metrics, also run:

```bash
go test ./internal/app ./internal/store/sqlite
```

Before completing implementation, run:

```bash
go test ./...
```

Run `go vet ./...` if the implementation changes exported domain/app structs or cross-package contracts.

## Formula Examples

Use tests or manual setup to verify these outcomes:

| Player levels | Monster XP total | Player count | Average PC level | Encounter level | Difference | Label |
|---------------|------------------|--------------|------------------|-----------------|------------|-------|
| 5, 5          | 0                | 2            | 5                | 1               | -4         | Trivial |
| 3, 3          | 20               | 2            | 3                | 1               | -2         | Simple |
| 2, 3          | 70               | 2            | 3                | 2               | -1         | Simple |
| 1, 1          | 60               | 2            | 1                | 2               | 1          | Average |
| 1, 1          | 80               | 2            | 1                | 3               | 2          | Hard |
| 1, 1          | 160              | 2            | 1                | 7               | 6          | Deadly |

## Manual UI Validation

1. Start the app with a test database:

   ```bash
   FALLOUT_TRACKER_DB_PATH=/tmp/fallout-difficulty-preview.db go run ./cmd/fallout-tracker
   ```

2. Create or open a campaign with at least two active player characters.
3. Open the encounter create dialog.
4. Load party members from the campaign.
5. Add a monster row with XP and quantity.
6. Change the monster quantity and confirm the difficulty preview updates immediately without pressing Save.
7. Remove the monster row and confirm the preview recalculates from the remaining draft rows.
8. Enter an invalid monster XP or quantity and confirm the preview shows Unknown/unavailable rather than a stale label.
9. Cancel the dialog and reopen the encounter list; confirm no encounter was created or updated only from preview edits.

## Expected Completion State

- The difficulty preview uses Trivial, Simple, Average, Hard, and Deadly labels.
- Easy and Normal no longer appear in the dynamic preview.
- Preview changes are local to the dialog until Save is chosen.
- No migration, sqlc generation, or database documentation regeneration is needed.
