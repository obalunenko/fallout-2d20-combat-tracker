# Quickstart: Existing Combat Tracker Baseline

## Purpose

Use this guide to validate the migrated baseline spec and planning artifacts against the current local application.

## Prerequisites

- Go toolchain compatible with `go.mod`.
- Project dependencies available through Go modules.
- macOS/Linux/Windows desktop environment for manual Fyne UI runs.
- Manual smoke validation should use tabletop-scale data: no more than 12 active encounter combatants and no more than 100 recent log entries in the DATA view for the migrated baseline.

## Automated Validation

Run the full baseline test suite:

```bash
go test ./...
```

Expected result:

- `internal/domain` tests pass for encounter ordering, resources, damage, healing, resistance, and difficulty behavior.
- `internal/app` tests pass for campaign, encounter, combat action, monster template, and operation log services.
- `internal/store/sqlite` tests pass for migrations, repository round trips, normalized stats, schema, and mappings.
- `internal/ui/fyneui` tests pass for collectors, presenters, formatters, controllers, active target, encounter order, and theme helpers.

For generated database changes in future specs, run:

```bash
make sqlc-generate
make db-doc-generate
go test ./...
```

For general code quality in future specs, run:

```bash
go vet ./...
make lint
make build
```

## Manual Local App Smoke Flow

Use an isolated database path for manual verification:

```bash
FALLOUT_TRACKER_DB_PATH="$(pwd)/tracker.db" make run
```

The startup smoke recorded below verifies application launch and migration startup with an isolated database. It does not replace the interactive click-through steps in this section.

Expected startup:

- The app opens a Fyne window titled "Fallout 2d20 Combat Tracker".
- If no active campaign exists, the UI prompts for campaign creation/opening.

## Campaign Flow

1. Create a campaign with a name, start date, and at least one player/character.
2. Confirm the CAMP view shows campaign overview and roster information.
3. Edit the campaign roster and mark a player inactive or change a character name.

Expected outcome:

- Campaign details persist.
- Active roster reflects active characters.
- Removed/inactive party characters are not retained in encounters after update.

## Encounter Flow

1. Create an encounter under the active campaign.
2. Add manual NPC combatants.
3. Load party members from the campaign roster.
4. Save NPCs as monster templates.
5. Reopen encounter creation/editing and load a monster template.

Expected outcome:

- Encounter order sorts by initiative descending.
- Party rows are linked to campaign characters.
- Monster templates can be reused as fresh NPC rows.
- Difficulty preview reflects party and enemy composition.

## Combat Flow

1. Launch an encounter.
2. Advance turns through at least one round wrap.
3. Add and spend Party AP.
4. Add and spend GM Threat.
5. Select a combatant and apply physical/energy/radiation/poison damage.
6. Heal a damaged or defeated combatant.

Expected outcome:

- Defeated combatants are skipped during turn advancement.
- Party AP caps at 6 and cannot be overspent.
- GM Threat cannot be overspent.
- Damage respects resistance, immunity, and torso-only behavior.
- Healing cannot exceed max HP and can clear defeated state.
- DATA tab shows operation log messages when log writes succeed.

## Persistence Flow

1. Close the app.
2. Reopen with the same `FALLOUT_TRACKER_DB_PATH`.
3. Open the campaign and encounter lists.

Expected outcome:

- Campaigns, encounters, party roster, monster templates, resources, HP state, and logs persist across restart.

## Known Manual Gaps

- There is no automated full-window Fyne click-through test for this smoke flow.
- Encounter difficulty scoring is specified in `spec.md` and summarized for users in `README.md`.
- Encounter log retention/export behavior is intentionally out of scope for the migrated baseline and remains a future spec candidate.

## Validation Log

- 2026-06-20: `go test ./internal/app -run TestBaselineSmokeFlowPersistsCampaignEncounterResourcesActionsAndLogs -count=1` passed. This automated smoke covers campaign creation, monster template save/list, encounter creation from linked party plus NPC, resource changes, damage/healing, operation logs, and reopening the same SQLite database.
- 2026-06-20: `go test ./...` passed for all packages. The macOS UI test link step emitted `ld: warning: ignoring duplicate libraries: '-lobjc'`; tests still passed.
- 2026-06-20: Startup smoke with isolated `FALLOUT_TRACKER_DB_PATH` passed. The app process started, applied Goose migrations through version 42, and accepted `TERM` after 5 seconds.
- Pending manual follow-up: Full interactive Fyne click-through for campaign/encounter/resource/damage/heal/reopen flows still requires a human at the desktop because this agent cannot verify window clicks visually.
