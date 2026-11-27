# Data Model: Assistant Control Of Campaigns And Encounters (MCP Server)

This feature adds no combat data. It adds three pieces of durable data and two in-memory structures, and it exposes existing entities through assistant operations. Existing entity definitions are in specs 001–003 and in `internal/domain`. This document covers only what is new or changed.

## Durable data (migration `00044_add_assistant_connection.sql`)

### Assistant Connection Settings (new table `assistant_connection_settings`)

There is a single row (`id = 1`, the same pattern as `app_state`). It holds the GM's connection choice so that it survives restarts (FR-003).

| Field | Type | Rules |
|---|---|---|
| `id` | INTEGER PK | `CHECK (id = 1)` |
| `enabled` | INTEGER (bool) | `CHECK (enabled IN (0, 1))`, default `0` (FR-003: off by default) |
| `port` | INTEGER | `CHECK (port BETWEEN 1024 AND 65535)`, default `47823`. The host is always `127.0.0.1` and is not stored. |
| `access_secret` | TEXT | Not blank. The migration does not generate it: the application generates 32 random bytes (base64url, no padding) on first read when the row is missing, and again on regeneration. |
| `created_at`, `updated_at` | DATETIME | Project convention: `STRFTIME('%Y-%m-%d %H:%M:%f','now')`, set explicitly on every write. |

Domain shape (`internal/domain/assistant.go`):

```text
AssistantConnectionSettings { Enabled bool; Port int; AccessSecret string }
```

- `Port` is validated as an integer from 1024 to 65535. Ports below 1024 need privileges, so they are rejected.
- `AccessSecret` never appears in logs, activity entries, errors, or `String()` output. Its type has a redacting `String`/`GoString`.

State transitions:

- Missing row → generated defaults (disabled, default port, new secret), saved on first read.
- `Enabled: false → true`: the server starts. On failure, `Enabled` stays `true` and the status shows the error, so the GM can change the port and retry.
- `RegenerateSecret`: a new secret is saved, and any connection using the old secret is refused from then on (FR-005).

### Assistant Activity Entry (new table `assistant_activity`)

| Field | Type | Rules |
|---|---|---|
| `id` | TEXT PK | UUID, generated in Go |
| `occurred_at` | DATETIME | ms precision, UTC |
| `operation` | TEXT | Not blank. The tool name, such as `apply_damage`, or `connection` for refused requests. |
| `entity_kind` | TEXT NULL | `CHECK (entity_kind IN ('campaign','player_character','encounter','combatant','monster_template'))` |
| `entity_id` | TEXT NULL | Identifier of the affected entity |
| `entity_label` | TEXT NOT NULL DEFAULT '' | Readable name at the time of the action, such as "Raider #2" |
| `outcome` | TEXT | `CHECK (outcome IN ('done','rejected','declined','timed_out','refused'))` |
| `detail` | TEXT NOT NULL DEFAULT '' | A short summary ("7 physical → torso, applied 4") or the rejection reason. Never contains the access secret. |
| `created_at`, `updated_at` | DATETIME | Project convention |

Index: `(occurred_at DESC)`.

Rules:

- **Retention (FR-024):** after each insert, delete everything except the newest 1,000 rows, in the same statement batch.
- **What is recorded:** only modifying and destructive tool calls, confirmation outcomes, and refused requests. Read-only tools are not recorded.
- **One entry per operation**, written for its final outcome. The `input_required` step of an in-conversation confirmation writes nothing, and the retry writes the entry. An unverifiable confirmation answer (forged, reused, or with different arguments) is recorded as `rejected`.
- **Refused requests:** refusals for the same reason (`connection_off` cannot happen because the listener is closed; `bad_secret`, `browser_origin`, `bad_host`) are collapsed to one entry per reason per minute, with `detail` counting the extra refusals. This stops a local process from flooding the log.
- **Failures:** an insert failure is a non-critical side effect (`runNonCriticalSideEffect`, new ID `audit.append_assistant_activity`). It never fails the tool call (Principle VII).

Domain shape:

```text
AssistantActivityEntry { ID string; OccurredAt time.Time; Operation string;
  EntityKind AssistantEntityKind; EntityID string; EntityLabel string;
  Outcome AssistantOutcome; Detail string }
```

### Encounter Log Entry — new `source` column (table `encounter_logs`)

| Field | Type | Rules |
|---|---|---|
| `source` | TEXT NOT NULL DEFAULT 'desktop' | `CHECK (source IN ('desktop','assistant'))`. Existing rows backfill to `desktop`. |

- `domain.EncounterLog` gains `Source OperationSource` (`desktop` | `assistant`).
- `appendOperationLog` takes the source from the request context (`app.SourceFromContext(ctx)`, which defaults to `desktop`).
- The DATA view shows assistant entries as `[timestamp] [R2] [AI] message` (FR-023).

A CHECK-constrained column is used here rather than a dictionary table because this is a two-value audit attribute, not a combat stat dimension. Principle III's preference for dictionaries covers combat stat dimensions. `docs/db-normalization.md` records this rationale.

### Migration summary

`00044_add_assistant_connection.sql`:

- **Up:** create `assistant_connection_settings`, create `assistant_activity` with its index, and `ALTER TABLE encounter_logs ADD COLUMN source ... DEFAULT 'desktop' CHECK(...)`.
- **Down:** drop both new tables. Rebuild `encounter_logs` without `source`, preserving rows and the existing indexes and constraints.
- **Then regenerate:** `internal/store/sqlite/sqlc/schema.sql`, `internal/store/sqlite/dbgen`, and `docs/db`.

## In-memory structures (no persistence)

### Operation Source (request context value)

`app.OperationSource` is `desktop` (default) or `assistant`. The MCP adapter sets `app.WithSource(ctx, app.SourceAssistant)` on every tool call. `internal/app` uses it for log tagging, change events, and revision bookkeeping. The desktop UI never sets it.

### Change Event (app → subscribers)

```text
ChangeEvent { Source OperationSource; Kinds []EntityKind; CampaignID string; EncounterID string }
```

- `Service` publishes one event after every successful mutating operation, after the write lock is released.
- Subscribers register with `Service.SubscribeChanges(func(ChangeEvent)) (unsubscribe func())`.
- Delivery is synchronous and non-blocking: subscribers must hand off work, for example to `fyne.Do`.
- The Fyne UI subscribes and handles only `Source == assistant`, because desktop actions already refresh. It coalesces bursts within 100 ms and calls `fyne.Do(refresher.Refresh)`. It also refreshes an open encounter list dialog, keeping the current selection (FR-021).

### Entity Revision (optimistic edit check, FR-022)

`Service` keeps `map[EntityKey]uint64` (`EntityKey{Kind, ID}`), guarded by the operation lock. Revisions reset on restart, which is safe because open editors do not survive a restart.

Revisions are bumped **only by assistant-sourced operations**, those whose context has `SourceFromContext(ctx) == assistant`. Desktop-sourced operations never bump them (Clarifications, 2026-11-27). The stale-edit warning therefore always means that the assistant made the change. Desktop dialogs are modal, so the desktop cannot change an entity while one of its own editors is open.

| Assistant operation | Revisions bumped |
|---|---|
| Create/Update/Restart encounter, advance turn, damage, heal | that encounter. Also the encounter's campaign when party combatants are present, because the encounter save writes linked character stats back to the player characters. |
| Delete encounter | that encounter, marked as deleted (tombstone) |
| Create/Update campaign | that campaign, and every encounter rewritten by `removeInactiveCampaignCharactersFromEncounters` / `syncEffectiveActiveEncounterPlayerCharacters` |
| Activate campaign/encounter, AP/Threat changes, monster template save | none (editors do not show this state) |

Editor protocol:

1. On open, the editor calls `svc.Revision(EntityKey)` and keeps the value.
2. On save, the command carries `ExpectedRevision`.
3. The service compares under the write lock and returns:
   - `domain.ErrStaleEdit` when the revision changed. The UI shows Reload / Overwrite; Overwrite resubmits with the current revision.
   - `domain.ErrEntityDeleted` when the entity has a tombstone. The UI shows a close-only notice.
4. `ExpectedRevision == 0` skips the check. The assistant always sends 0 (last write wins for the assistant); only desktop editors send a value.

### Pending Confirmation (MCP adapter)

```text
ConfirmationRequest { Operation string; EntityKind; EntityID; EntityLabel; Summary string }
Decision = accepted | declined | timed_out
```

- Each request is shown either through client elicitation or, as a fallback, through a desktop dialog. Desktop dialogs are shown one at a time (FIFO), and each has its own 50-second deadline from enqueue. A request that expires while still queued is dropped without being shown and returns `timed_out`.
- An elicitation's `requestState` token expires 2 minutes after it is issued. An authentic expired token returns `timed_out`. A forged or reused token, or one used with different arguments, returns `declined` and is recorded as `rejected` (FR-015).
- The request content comes from stored state, such as the entity name read before confirmation, never from text the assistant supplies, so a request cannot pre-answer or mislabel its confirmation.

## Exposed views of existing entities (tool results)

Tool results map domain values onto stable, JSON-friendly DTOs in `internal/mcpserver`. See `contracts/mcp-tools.md` for the schemas. Mapping notes:

| Domain | Identifier exposed | Notes |
|---|---|---|
| `Campaign` | `campaign_id` | `is_active` is derived by comparing with `GetActiveCampaign`. `party_ap` and `gm_threat` come from `Campaign.Resources`. |
| `NewCampaignPlayer` | `character_id` (= `Character.ID` = `player_characters.id`) and `player_name` | The repository matches players by case-insensitive player name. Renaming a character creates a new character ID, which the contract documents. |
| `EncounterSummary` | `encounter_id` | Includes the difficulty metrics computed on read. |
| `Encounter` / `Combatant` | `encounter_id` / `combatant_id` | Includes `turn_index`, `active_combatant_id`, per-location DR, immunities, and `defeated`. |
| `Combatant` (monster template) | `template_id` | Templates are deduplicated by case-insensitive name on save. |
| `EncounterLog` | — | `round`, `message`, `created_at`, `source` |
| `AssistantActivityEntry` | `activity_id` | Read-only for the assistant |

## Validation rules reused (not reimplemented)

- Combatant validation uses `domain.ValidateCombatant`. Campaign player preparation uses `prepareCampaignPlayers`. Encounter combatant preparation uses `prepareEncounterCombatants` and `resolveLinkedPartyCombatants`. Damage and healing go through `(*Encounter).ApplyDamage` / `Heal`. AP and Threat go through `Resources`. Difficulty uses `domain.EvaluateEncounterDifficulty`.
- The adapter validates only the wire format: JSON schema, enum values for damage type and body location, and integer ranges declared in the schema. Business rules are rejected by the application layer, and the adapter maps those errors to tool errors (FR-017).
