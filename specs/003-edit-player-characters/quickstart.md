# Quickstart: Edit Campaign Player Characters

## Prerequisites

- Repository root: `fallout`
- Go toolchain matching `go.mod`
- Existing local prerequisites for Fyne and SQLite generation tools

## Generate Persistence Artifacts

After adding migration `00043` and updating `internal/store/sqlite/sqlc/query.sql`, regenerate the schema and typed database access:

```bash
make sqlc-generate
```

Regenerate database documentation:

```bash
make db-doc-generate
```

Confirm generated files have no unexpected manual-only changes before continuing.

## Focused Automated Validation

Run domain and application behavior first:

```bash
go test ./internal/domain ./internal/app
```

Run persistence and migration coverage:

```bash
go test ./internal/store/sqlite
```

Run UI collector, formatter, and refresh coverage:

```bash
go test ./internal/ui/fyneui
```

Before completing implementation, run the full gates:

```bash
go test ./...
go vet ./...
make lint
make build
```

## Automated Scenarios

Tests should prove at least these cases:

1. Migration `00043` upgrades an existing database, preserves existing player characters, sets notes to blank, and creates seven value-1 S.P.E.C.I.A.L. rows per character.
2. Migration Down removes only the feature’s new storage and leaves earlier campaign/player-character data valid.
3. Campaign creation without explicit character details stores blank notes and seven default values of 1.
4. A valid campaign edit round-trips exact multiline notes, all seven S.P.E.C.I.A.L. values, level, HP, Defense, DR, and immunity.
5. Invalid S.P.E.C.I.A.L., HP, Defense, or DR values are rejected without persistence.
6. An injected repository failure rolls back notes, S.P.E.C.I.A.L., combat stats, and linked active-combatant changes together.
7. Editing an active linked character updates its active encounter snapshot and defeated state without changing round, turn, position, or unrelated combatants.
8. The same edit leaves a closed encounter’s scalar and DR snapshot unchanged when reopened.
9. Cancel and validation-error UI paths retain the expected saved/draft values and do not call persistence incorrectly.

## Manual UI Validation

1. Create an isolated temporary database directory and start the app:

   ```bash
   FALLOUT_EDIT_TEST_DIR=$(mktemp -d /tmp/fallout-character-edit.XXXXXX)
   FALLOUT_TRACKER_DB_PATH="$FALLOUT_EDIT_TEST_DIR/tracker.db" go run ./cmd/fallout-tracker
   ```

2. Create a campaign with at least two player characters.
3. Open Campaigns, select the campaign, and choose Edit.
4. Expand one character’s details and enter multiline notes plus distinct values for all seven S.P.E.C.I.A.L. fields.
5. Change level, current/max HP, Defense, poison DR, and at least one body-location DR value. Save and reopen the campaign editor; verify every value is unchanged.
6. Enter current HP greater than maximum HP and an invalid S.P.E.C.I.A.L. value. Verify Save shows field-specific feedback, keeps draft text, and does not change the main view.
7. Correct the fields, then Cancel. Reopen and verify the canceled values were not stored.
8. Create two encounters containing the edited player character. Activate the older encounter, then reactivate the intended current encounter so activation order is explicit.
9. Edit the character again and save. Verify the active encounter immediately shows the new level, HP, Defense, DR, and defeated state while retaining round and turn.
10. Open the other encounter and verify its stored character snapshot did not change.
11. Close and restart the app using the same test database; verify notes and all S.P.E.C.I.A.L. values persist.

## Expected Completion State

- Campaign editing supports notes and all seven S.P.E.C.I.A.L. values without making the roster table unusably wide.
- Existing combat fields and DR remain editable and validated.
- Valid edits are atomic and persist across restart; invalid or canceled edits persist nothing.
- The active linked combatant matches campaign combat values after save.
- Closed encounters retain their stored combatant snapshots.
- Schema, generated database access, and database documentation match migration `00043`.

## Implementation Validation Record

- Automated persistence coverage confirms exact multiline-note and seven-value S.P.E.C.I.A.L. round trips.
- Automated encounter coverage confirms campaign edits synchronize the effective active snapshot and leave a closed snapshot unchanged.
- `make sqlc-generate` completed successfully with migration `00043`.
- Final automated gates on 2026-08-08: `make sqlc-generate`, `make db-doc-generate`, focused layer tests, `go test ./...`, `go vet ./...`, `make lint` (0 issues), and `make build` all completed successfully. The macOS linker emitted its existing duplicate `-lobjc` warning during Fyne test/build linking.
- The desktop workflow still requires a human visual pass for SC-001 (edit completion within two minutes) and SC-005 (clear feedback within one save attempt); automated widget and validation tests cover the underlying payload and error behavior but cannot measure human usability.
