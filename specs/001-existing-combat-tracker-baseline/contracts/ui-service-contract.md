# UI And Service Contracts: Existing Combat Tracker Baseline

## Contract Scope

This project does not expose an HTTP API or public library API. The relevant contracts are the user-facing desktop workflows and the application service methods used by Fyne UI code.

## Campaign Contract

**UI entry points**: CAMP tab, header campaign actions, campaign editor dialog, campaign list dialog.

**Service operations**:

- `CreateCampaign(ctx, id, name, startDate, players)`
- `UpdateCampaign(ctx, campaignID, name, startDate, players)`
- `ListCampaigns(ctx)`
- `ListCampaignPlayers(ctx, campaignID)`
- `ActivateCampaign(ctx, campaignID)`
- `GetActiveCampaign(ctx)`

**Preconditions**:

- Campaign name, start date, and at least one player are required for create/update.
- Player name and character name are required.

**Postconditions**:

- Created campaign persists with players and player characters.
- First campaign becomes active when no active campaign exists.
- Updating a campaign preserves or versions player characters according to active character identity.
- Inactive/removed campaign characters are removed from existing encounters.

## Encounter Contract

**UI entry points**: NEW ENCOUNTER, OPEN ENCOUNTER, encounter editor/list/order views.

**Service operations**:

- `CreateEncounter(ctx, id, name, combatants)`
- `UpdateEncounter(ctx, encounterID, name, combatants)`
- `ListEncounters(ctx)`
- `GetEncounter(ctx)`
- `GetEncounterByID(ctx, encounterID)`
- `ActivateEncounter(ctx, encounterID)`
- `RestartEncounter(ctx, encounterID)`
- `DeleteEncounter(ctx, encounterID)`
- `AdvanceTurn(ctx)`

**Preconditions**:

- Active campaign exists before encounter create/list flows.
- Encounter name and at least one valid combatant are required.
- Encounter ID is required for update/activate/restart/delete/get-by-id.

**Postconditions**:

- Combatants are sorted by initiative descending, with name tie-breaks.
- Active combatant follows turn index.
- Advance turn skips defeated combatants and increments round after wrap.
- Restart resets NPC HP/defeated state while preserving campaign resources.
- Delete is soft-delete.

## Combat Action Contract

**UI entry points**: active target panel, apply damage dialog, heal dialog, resource buttons.

**Service operations**:

- `AddPartyAP(ctx, value)`
- `SpendPartyAP(ctx, value)`
- `AddThreat(ctx, value)`
- `SpendThreat(ctx, value)`
- `ExecuteApplyDamage(ctx, ApplyDamageCommand)`
- `ExecuteHeal(ctx, HealCommand)`

**Preconditions**:

- Active encounter exists.
- Combatant ID is required for damage/heal.
- Damage/heal amount cannot be negative.
- Damage type and body location must be known.

**Postconditions**:

- Party AP remains between 0 and 6.
- GM Threat remains at least 0.
- Effective damage is `max(raw - resistance, 0)` unless immune, in which case it is 0.
- Damage to torso-only combatants uses torso resistance.
- Healing cannot exceed max HP and clears defeated state when HP becomes positive.
- Party character state is synchronized for linked party combatants where persistence supports it.

## Monster Template Contract

**UI entry points**: encounter editor "Save NPCs To DB" and "Load Monster From DB".

**Service operations**:

- `SaveMonsterTemplates(ctx, monsters)`
- `ListMonsterTemplates(ctx)`

**Preconditions**:

- At least one monster is required.
- Monster name is required.
- Monster level must be at least 1.

**Postconditions**:

- Saved templates are NPC-only.
- Player character links, active state, and defeated state are cleared.
- Duplicate names in one save batch are ignored after the first.
- Existing database template with the same name is updated rather than duplicated.

## Persistence Contract

**Startup operations**:

- Resolve DB path from `FALLOUT_TRACKER_DB_PATH` or default config path.
- Open SQLite with foreign keys enabled.
- Run embedded Goose migrations before repository use.

**Failure and recovery behavior**:

- DB path resolution, database open, or migration failure prevents application startup and returns/logs a startup error.
- Repository operations that run inside transactions roll back partial writes when the transaction fails.
- Campaign-scoped operations require an active campaign.
- Encounter-scoped operations require an active encounter or a valid encounter ID, depending on the operation.

**Generation contract**:

- `internal/store/sqlite/sqlc/schema.sql` reflects migrated schema.
- `internal/store/sqlite/sqlc/query.sql` defines typed query source.
- `internal/store/sqlite/dbgen` is generated from sqlc and not manually edited.
- `docs/db` is regenerated when schema changes.

## Operation Log Contract

**Service operations**:

- `ListEncounterLogs(ctx, encounterID)`
- Non-public append path used by combat/resource/encounter service operations.

**Log-producing operations**:

- Encounter created.
- Encounter activated.
- Encounter updated.
- Encounter restarted.
- Turn advanced.
- Party AP changed.
- GM Threat changed.
- Damage applied.
- Healing applied.

**Preconditions**:

- Encounter ID and message are required for append.

**Postconditions**:

- Major operations append round-stamped messages when possible.
- Append failure is logged and counted as a non-critical side effect.
- Primary saved combat/resource action is not failed solely because log append failed.

**Retention and export**:

- The migrated baseline defines logs as append-only.
- Retention, export, compaction, and cleanup are intentionally deferred to a future log-management spec.
