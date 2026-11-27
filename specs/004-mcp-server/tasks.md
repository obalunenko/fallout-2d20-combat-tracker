---

description: "Task list for 004-mcp-server: assistant control of campaigns and encounters"
---

# Tasks: Assistant Control Of Campaigns And Encounters (MCP Server)

**Input**: Design documents from `/specs/004-mcp-server/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (mcp-tools.md, mcp-connection.md, desktop-ui.md), quickstart.md

**Tests**: Required. Constitution Principle IV says "Tests Guard Every Behavioral Change", and the spec's Constitution Check lists the test impact for each layer. Each story writes its tests first and checks that they fail before implementing.

**Organization**: Tasks are grouped by user story. The P1 stories (US1, US2, US3, US6) come first in spec order, then US4 (P2), then US5 (P3). The Claude Desktop bridge (FR-008) and the gates are cross-cutting final phases.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (US1–US6), used only in story phases
- Paths are repository-relative. See the `plan.md` structure tree.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Governance amendment, the new dependency, the package skeleton, and a check of the main protocol risk.

- [x] T001 Amend `.specify/memory/constitution.md` to v1.2.0, following research R18:
  - **Principle I diagram**: add `cmd/fallout-tracker -> internal/mcpserver -> internal/app`.
  - **Principle VI**: add an exception for "a loopback-only (`127.0.0.1`), off-by-default, access-secret-protected assistant endpoint embedded in the desktop app; no internet access, remote hosting, or accounts".
  - **Project Structure Rules**: add `- MCP adapter: internal/mcpserver (may import internal/app, internal/domain, and the MCP SDK; never internal/store or internal/ui).`
  - **Version line**: set `**Version**: 1.2.0 | ... | **Last Amended**: <today>` and add a one-line rationale that cites `specs/004-mcp-server`.
  - **`CLAUDE.md`**: in the hand-written "Project Boundaries" section, outside the SPECKIT markers, add the bullet `` `internal/mcpserver`: MCP adapter (tools, resources, skills, admission, stdio bridge). It may depend on `internal/app` and `internal/domain`, but not on `internal/store` or `internal/ui`. ``
- [x] T002 Add `github.com/modelcontextprotocol/go-sdk@v1.8.0` and promote `golang.org/x/sync` to a direct requirement in `go.mod`/`go.sum`:
  1. `go get github.com/modelcontextprotocol/go-sdk@v1.8.0 golang.org/x/sync@v0.20.0`
  2. `go mod tidy`
  3. `go mod verify`
- [x] T003 [P] Create `internal/mcpserver/doc.go`. Its package comment states the adapter's role and its allowed imports (`internal/app`, `internal/domain`, `github.com/modelcontextprotocol/go-sdk/...`). It forbids `internal/store/...` and `internal/ui/...`.
- [x] T004 Write the protocol-compatibility spike test `internal/mcpserver/protocol_compat_test.go`:
  - Build an `mcp.NewStreamableHTTPHandler(..., &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})` with one dummy tool behind `httptest.NewServer`.
  - **Assert 1**: a go-sdk client that negotiates `2026-07-28` can call the tool and sees `req.ProtocolVersion() == "2026-07-28"`.
  - **Assert 2**: a client limited to `2025-11-25` (via `ClientOptions`/supported versions) can `initialize` and call the tool without a session.
  - **Assert 3**: `GET /mcp` returns 405.
  - If assertion 2 fails, record the finding in `specs/004-mcp-server/research.md` R8 and stop for re-planning before Phase 2.
  - **Manual header capture**, needed before T032:
    1. Add a test helper `-run TestCaptureClientHeaders`, skipped unless `FALLOUT_CAPTURE_HEADERS=1`. It serves that handler on `127.0.0.1:${FALLOUT_CAPTURE_PORT:-47899}`, a separate port so a running tracker on 47823 does not conflict. It logs every request's headers, with `Authorization` redacted.
    2. Connect a real Claude Code (v2.1.331 or later) with `claude mcp add --transport http fallout-capture http://127.0.0.1:47899/mcp --header "Authorization: Bearer capture"`, run `claude mcp list`, then `claude mcp remove fallout-capture`.
    3. Record in research R3 whether it sends `Origin` or `Sec-Fetch-*`, and their values.
    4. If it sends an `Origin` other than the loopback origin, stop and re-plan R3.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Everything any story needs:

- domain types and errors;
- migration `00044` and the repositories;
- the operation gate, source context, change events and revisions in `internal/app`;
- the assistant ports and settings service;
- the MCP server core with request admission and connection lifecycle;
- a minimal desktop toggle so the endpoint can be started.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Domain

- [x] T005 [P] Add sentinel errors to `internal/domain/errors.go`:
  - `ErrTrackerBusy`: "tracker busy, try again"
  - `ErrStaleEdit`: "entity was changed after the editor opened"
  - `ErrEntityDeleted`: "entity was deleted"
- [x] T006 [P] Create `internal/domain/assistant.go`:
  - `OperationSource` (`"desktop"` default, `"assistant"`)
  - `AssistantOutcome` with the values exactly `'done','rejected','declined','timed_out','refused'`
  - `AssistantEntityKind` with the values exactly `'campaign','player_character','encounter','combatant','monster_template'`
  - `DefaultAssistantPort = 47823`
  - `AssistantConnectionSettings{Enabled bool; Port int; AccessSecret AccessSecret}` with `Validate()`, which enforces "`CHECK (port BETWEEN 1024 AND 65535)`" and a non-blank secret
  - `AccessSecret` string type whose `String()`/`GoString()` return `"[redacted]"`
  - `AssistantActivityEntry{ID; OccurredAt time.Time; Operation; EntityKind; EntityID; EntityLabel; Outcome; Detail}`
- [x] T007 [P] Write `internal/domain/assistant_test.go`:
  - port boundaries: 1023 rejected, 1024 and 65535 accepted, 65536 rejected;
  - a blank secret is rejected;
  - `fmt.Sprintf("%v %#v", s, s)` never contains the secret;
  - the enum values match the CHECK lists in T006.
- [x] T008 Add `Source OperationSource` to `domain.EncounterLog` in `internal/domain/encounter.go` (zero value treated as `desktop`).

### Persistence (migration 00044)

- [x] T009 Create `internal/store/sqlite/migrations/00044_add_assistant_connection.sql`, per `data-model.md`.
  - **Up**, table `assistant_connection_settings`:
    - `id INTEGER PRIMARY KEY CHECK (id = 1)`
    - `enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1))`
    - `port INTEGER NOT NULL DEFAULT 47823 CHECK (port BETWEEN 1024 AND 65535)`
    - `access_secret TEXT NOT NULL CHECK (trim(access_secret) <> '')`
    - `created_at`/`updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f','now'))`
    - No seed row.
  - **Up**, table `assistant_activity`:
    - `id TEXT PRIMARY KEY`
    - `occurred_at DATETIME NOT NULL`
    - `operation TEXT NOT NULL CHECK (trim(operation) <> '')`
    - `entity_kind TEXT NULL CHECK (entity_kind IN ('campaign','player_character','encounter','combatant','monster_template'))`
    - `entity_id TEXT NULL`
    - `entity_label TEXT NOT NULL DEFAULT ''`
    - `outcome TEXT NOT NULL CHECK (outcome IN ('done','rejected','declined','timed_out','refused'))`
    - `detail TEXT NOT NULL DEFAULT ''`
    - `created_at`/`updated_at`
    - Index `idx_assistant_activity_occurred (occurred_at DESC)`.
  - **Up**, column: `ALTER TABLE encounter_logs ADD COLUMN source TEXT NOT NULL DEFAULT 'desktop' CHECK (source IN ('desktop','assistant'))`.
  - **Down**: drop both tables. Rebuild `encounter_logs` without `source`, keeping rows, the FK `ON DELETE CASCADE`, the `round >= 1` and non-blank message checks, and `idx_encounter_logs_encounter_created`.
- [x] T010 Add queries to `internal/store/sqlite/sqlc/query.sql`:
  - **Settings**: `GetAssistantConnectionSettings`, `UpsertAssistantConnectionSettings` (`ON CONFLICT(id) DO UPDATE`, sets `updated_at` explicitly).
  - **Activity**:
    - `InsertAssistantActivity`
    - `PruneAssistantActivity`: `DELETE ... WHERE id NOT IN (SELECT id ... ORDER BY occurred_at DESC, rowid DESC LIMIT 1000)`
    - `ListAssistantActivity(limit)`: newest first
    - `ClearAssistantActivity`
  - **Encounter logs**: add `source` to `InsertEncounterLog` and to the `ListEncounterLogsByEncounterID` select list.
- [x] T011 Run `make sqlc-generate`. It regenerates `internal/store/sqlite/sqlc/schema.sql` and `internal/store/sqlite/dbgen/`, which must never be hand-edited.
- [x] T012 [P] Add `_pragma=busy_timeout(5000)` to both DSN forms in `internal/store/sqlite/db.go` (`:97` and `:105`), next to `foreign_keys(1)`.
- [x] T013 Create `internal/store/sqlite/assistant_repository.go` on `*EncounterStore`:
  - `GetAssistantConnectionSettings(ctx) (domain.AssistantConnectionSettings, bool, error)`. The `bool` reports whether the row was found.
  - `SaveAssistantConnectionSettings(ctx, s)`
  - `AppendAssistantActivity(ctx, e)`: insert plus prune, in one `runInTx`
  - `ListAssistantActivity(ctx, limit)`
  - `ClearAssistantActivity(ctx)`
- [x] T014 Thread the log source through the store:
  - `AppendEncounterLog(ctx, encounterID string, round int, message string, source domain.OperationSource)` writes `source` (empty means `desktop`) in `internal/store/sqlite/encounter_logs_repository.go`.
  - The source is mapped in `ListEncounterLogs` and `internal/store/sqlite/mappers.go`.
- [x] T015 [P] Write `internal/store/sqlite/assistant_repository_test.go`:
  - **Settings**: a missing row returns `found=false`, and save/load round-trips.
  - **Activity**: insert 1,001 entries and list returns exactly the newest 1,000. Clear empties the table.
  - **CHECKs**: an invalid `outcome` or `entity_kind` is rejected.
  - **Log source**: round-trips, and existing rows read as `desktop`.
- [x] T016 [P] Extend the migration tests in `internal/store/sqlite/migration_test.go` (create it if missing, following `test_helpers_test.go`):
  - `00044` up, then down, then up on a DB that has encounter logs. Rows survive, and backfilled `source = 'desktop'`.
  - After Down, `encounter_logs` keeps its FK, checks and index.
- [x] T017 Run `make db-doc-generate` to refresh `docs/db/`. Add a section to `docs/db-normalization.md` explaining why `encounter_logs.source` is a CHECK-constrained column (a two-value audit attribute, not a combat stat dimension) and why `assistant_connection_settings` is a single row like `app_state`.

### Application core

- [x] T018 Extend `internal/app/repository_contracts.go`:
  - add `AssistantSettingsRepository` (get/save) and `AssistantActivityRepository` (append/list/clear);
  - change `EncounterLogRepository.AppendEncounterLog` to take `source domain.OperationSource`;
  - embed both new interfaces in `EncounterRepository`;
  - update every hand-written fake: `logFailingRepo` and `contextCaptureRepo` in `internal/app/service_test.go`, and `refresherRepo` in `internal/ui/fyneui/main_view_refresher_test.go`.
- [x] T019 [P] Create `internal/app/source.go` with `WithSource(ctx, domain.OperationSource) context.Context` and `SourceFromContext(ctx) domain.OperationSource` (default `desktop`). Make `appendOperationLog` in `internal/app/operation_log.go` pass `SourceFromContext(ctx)` to `AppendEncounterLog`.
- [x] T020 Create `internal/app/gate.go`, the operation gate:
  - `semaphore.NewWeighted(gateWeight)`;
  - `acquireRead(ctx)` takes weight 1, `acquireWrite(ctx)` takes the full weight;
  - a failed acquire under a deadline returns `domain.ErrTrackerBusy`.
  Add the gate field to `Service` in `internal/app/service.go`, initialized in `NewServiceWithLogfAndTimeout`.
- [x] T021 Wrap every exported `*Service` method in the gate, after `contextForOperation`:
  - read methods take the read gate; mutating methods take the write gate;
  - files: `internal/app/campaign_service.go`, `internal/app/encounter_service.go`, `internal/app/combat_action_service.go`, `internal/app/monster_template_service.go`, `internal/app/operation_log.go`;
  - each wrapper pair (`CreateCampaign`→`ExecuteCreateCampaign`, `UpdateCampaign`, `CreateEncounter`, `UpdateEncounter`, `ApplyDamage`, `Heal`, `SaveMonsterTemplates`) acquires once, in the `Execute*` method;
  - internal helpers (`resolveLinkedPartyCombatants`, `updateActiveEncounterResources`, and so on) never call gated methods.
- [x] T022 Create `internal/app/changes.go`:
  - `ChangeEvent{Source domain.OperationSource; Kinds []domain.AssistantEntityKind; CampaignID, EncounterID string}`
  - `(*Service).SubscribeChanges(func(ChangeEvent)) (unsubscribe func())`, with subscribers guarded by their own mutex
  - `publish` is called after the write gate is released, by every successful mutating method in the T021 files
- [x] T023 Create `internal/app/revisions.go`:
  - `EntityKey{Kind domain.AssistantEntityKind; ID string}`, `(*Service).Revision(EntityKey) uint64`, `bump`, `tombstone`;
  - the bump rules exactly as in the `data-model.md` "Entity Revision" table;
  - bump **only** when `SourceFromContext(ctx) == domain.SourceAssistant`; desktop-sourced operations never bump.
  Add `ExpectedRevision uint64` to `UpdateCampaignCommand` and `UpdateEncounterCommand` in `internal/app/commands.go`. Under the write gate:
  - a non-zero value that differs returns `domain.ErrStaleEdit`;
  - a tombstoned entity returns `domain.ErrEntityDeleted`;
  - `0` skips the check.
- [x] T024 [P] Write `internal/app/gate_test.go`. Run it with `-race`, against a temp-dir SQLite service via `newSQLiteService`:
  - 50 goroutines × `ApplyDamage(1)` in parallel with 20 `AdvanceTurn`: final HP = start − applied total, and round/turn are consistent;
  - no `database is locked`;
  - a write held by the test makes another write with a 50 ms deadline return `ErrTrackerBusy`;
  - every wrapper pair completes without deadlock (each call under a 1 s timeout).
- [x] T025 [P] Write `internal/app/changes_test.go` and `internal/app/revisions_test.go`:
  - **Events**: one event per successful mutation, none on failure, published after release (a subscriber can call a read method without deadlocking), with the source taken from the context.
  - **Revisions**: the bump rules per operation; the same operations with a desktop source leave revisions unchanged; stale and deleted detection; `0` skips the check.
  - **Log source**: extend `internal/app/operation_log_test.go` to assert that an assistant-sourced log entry is stored with `source = assistant`.
- [x] T026 Create `internal/app/assistant_ports.go`:
  - `ConnectionStatus{State ("stopped"|"running"|"failed"); Address string; Reason string}`
  - `AssistantConnection` interface: `StartIfEnabled(ctx) error`, `Start(ctx) error`, `Stop(ctx) error`, `Status() ConnectionStatus`, `Apply(ctx, domain.AssistantConnectionSettings) error`, `OnStatusChange(func(ConnectionStatus))`
  - `ConfirmationRequest{Operation string; EntityKind domain.AssistantEntityKind; EntityID, EntityLabel, Summary string}`
  - `ConfirmationDecision` (accepted/declined/timed_out)
  - `DesktopConfirmer` interface: `Confirm(ctx, ConfirmationRequest) (ConfirmationDecision, error)`
  - `ConfirmationRouter`: `SetDesktop(DesktopConfirmer)`, and `Confirm` returns declined while nothing is registered
- [x] T027 Create `internal/app/assistant_service.go`:
  - `GetAssistantSettings(ctx)`: when no row exists, generate a 32-byte `crypto/rand` secret encoded with `base64.RawURLEncoding`, with defaults `Enabled=false, Port=47823`, and save it;
  - `SaveAssistantSettings(ctx, s)`: validates the settings;
  - `RegenerateAccessSecret(ctx)`;
  - `AppendAssistantActivity(ctx, e)`: a non-critical side effect via `runNonCriticalSideEffect` with the new ID `audit.append_assistant_activity` (add it to the consts in `internal/app/service.go`);
  - `ListAssistantActivity(ctx, limit 1..200)`;
  - `ClearAssistantActivity(ctx)`;
  - **Gate** (T020/T021):
    - `GetAssistantSettings`, the lazy-create path included, `SaveAssistantSettings`, `RegenerateAccessSecret`, `AppendAssistantActivity` and `ClearAssistantActivity` take the write gate;
    - `ListAssistantActivity` takes the read gate;
    - none of them publishes a change event or bumps revisions;
    - the admission middleware calls `AppendAssistantActivity` outside any other gate hold.
- [x] T028 [P] Write `internal/app/assistant_service_test.go`:
  - a fresh database yields `Enabled == false` and `Port == 47823` (US6 acceptance scenario 1, FR-003);
  - lazy generation happens once, and a second read returns the same secret;
  - regeneration changes the secret;
  - an invalid port is rejected;
  - a failing activity append increments the side-effect counter and returns no error;
  - the limit bounds are enforced.

### MCP server core

- [x] T029 [P] Create `internal/mcpserver/errors.go`: map errors to tool errors per the `contracts/mcp-tools.md` "Error contract":
  - `invalid_input`, `not_found` (`ErrEncounterNotFound`, `ErrCampaignNotFound`, unknown IDs), `no_campaign`, `no_active_encounter`, `busy`, `declined`, `timed_out`, `internal`;
  - each becomes `isError: true` with `structuredContent {"error":{"code","message","field"?}}` and one text block;
  - `field` is optional (`omitempty`). It is set only for schema violations and the adapter's own checks (`quantity`, `delta`, `limit`, `template_id`/`monster` exclusivity). Domain validation errors pass through without it, because their message already names the field;
  - `internal` errors return a generic message and log the detail.
- [x] T030 [P] Create `internal/mcpserver/dto.go` with all the shared DTOs and their mappings from domain values, as listed in `contracts/mcp-tools.md` "Shared DTOs": `DR`, `Resistance`, `Immunity`, `Special`, `Campaign`, `Character`, `Combatant`, `Difficulty`, `EncounterSummary`, `Encounter`, `MonsterTemplate`, `LogEntry`, `Activity`, `CharacterInput`, `MonsterInput`, `DraftMonster`.
  - JSON names are snake_case.
  - Schema tags: `jsonschema` `minimum` constraints, and enums for `damage_type`, `location`, `side` and `source`.
- [x] T031 Create `internal/mcpserver/activity.go`: `recordActivity(ctx, svc, op, kind, id, label, outcome, detail)`. Its detail strings never include the secret.
- [x] T032 Create `internal/mcpserver/admission.go`, the middleware chain from `contracts/mcp-connection.md` "Request admission":
  1. `refuseBrowser`: 403 if `Origin` is present and is not exactly `http://127.0.0.1:<port>` or `http://localhost:<port>`, or if `Sec-Fetch-Site` is present with a value other than `none` or `same-origin`. Adjust this per the T004 header capture recorded in research R3.
  2. `requireLoopbackHost`: 403 unless `Host` is `127.0.0.1:<port>` or `localhost:<port>`.
  3. `auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})`, where the verifier compares against an `atomic.Pointer[string]` secret with `subtle.ConstantTimeCompare`.
  Every refusal records an activity entry `refused` with reason `browser_origin`, `bad_host` or `bad_secret`, collapsed per reason per minute (a counter goes into `detail`).
- [x] T033 Create `internal/mcpserver/server.go`:
  - `newServer(svc *app.Service, deps)` returns `*mcp.Server` with `Implementation{Name: "fallout-tracker", Title: "Fallout 2d20 Combat Tracker", Version: deps.Version}` (the version comes from T040, research R21), `ServerOptions{Instructions, Capabilities, SetCacheable}` (`Instructions` starts as a placeholder; T080 writes the final text);
  - `addTool[In, Out]` sets **all four** annotations explicitly (`ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint=false`), wraps ctx with `app.WithSource(ctx, domain.SourceAssistant)`, and maps errors through T029;
  - the handler is built with `mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true})`.
- [x] T034 Create `internal/mcpserver/connection.go`, `Connection`, implementing `app.AssistantConnection`:
  - `StartIfEnabled` reads `svc.GetAssistantSettings`;
  - `Start` runs `net.Listen("tcp", "127.0.0.1:<port>")`. On error the status is `failed` with "port N is in use" and the error is returned without crashing (FR-006);
  - `http.Server` mux: `POST /mcp` goes through the admission chain to the handler, `GET`/`DELETE /mcp` return 405, other paths 404;
  - `Stop` calls `Shutdown` with a 5 s deadline;
  - `Apply` restarts on a port change and atomically swaps the secret on secret change;
  - `OnStatusChange` callbacks fire on every transition.
- [x] T035 [P] Create the test helpers in `internal/mcpserver/testutil_test.go`:
  - a temp-dir SQLite `app.Service` with a seeded campaign (mirror `internal/app/service_test.go:47-57`);
  - a running `Connection` on port 0 / `httptest`;
  - a go-sdk client with `StreamableClientTransport` and a bearer header;
  - an option to declare or omit form elicitation;
  - a fake `DesktopConfirmer` with a programmable decision;
  - a `callTool` helper that returns the structured content.
- [x] T036 [P] Write `internal/mcpserver/admission_test.go`. These assert SC-008:

  | Request | Expected |
  |---|---|
  | No token | 401 |
  | Wrong token | 401 |
  | `Origin: http://evil.example` | 403 |
  | `Origin: http://127.0.0.1:8080` (another loopback port) | 403 |
  | `Origin: http://127.0.0.1:<port>` with a valid token | 200 |
  | `Sec-Fetch-Site: cross-site` | 403 |
  | `Sec-Fetch-Site: same-origin` with a valid token | 200 |
  | `Host: evil.example:47823` | 403 |
  | Valid token | 200 |
  | Secret rotated, old token | 401 |
  | Connection stopped | connection refused |

  Also assert that the activity entries are collapsed (5 bad-secret requests within a minute produce 1 entry).
- [x] T037 [P] Write `internal/mcpserver/connection_test.go`:
  - start on an occupied port gives `failed` and a reason, and the service still works;
  - Stop during an in-flight call: the call finishes or is cancelled, and later calls are refused;
  - `Apply` with a new port moves the listener;
  - the listener address is `127.0.0.1` after `Start` and after `Apply`, never `0.0.0.0` or `::` (SC-008, "not from another computer");
  - the status callbacks fire.

### Desktop wiring (minimal, so the endpoint can be enabled)

- [x] T038 Add `Options{Assistant app.AssistantConnection; Confirmations *app.ConfirmationRouter}` to `fyneui.Run` in `internal/ui/fyneui/app.go`. Once the window is built, call `opts.Assistant.StartIfEnabled(ctx)` (errors go to `handleErr`, not fatal). In `internal/ui/fyneui/shutdown.go`, call `opts.Assistant.Stop` with a 5 s timeout before `fyne.Do(app.Quit)`.
- [x] T039 Add a header button labelled `ASSISTANT · OFF|ON|ERROR` (updated through `OnStatusChange` and `fyne.Do`) in `internal/ui/fyneui/main_screen.go` and `internal/ui/fyneui/layout.go`. Create `internal/ui/fyneui/assistant_connection_dialog.go` with:
  - an Enable switch: save the settings, then `Start`/`Stop`. On failure the switch stays on and the error shows;
  - a status line (`Running at http://127.0.0.1:<port>/mcp` | `Stopped` | `Failed: <reason>`);
  - the access secret, masked, with Show and Copy.
- [x] T040 Wire everything in `cmd/fallout-tracker/main.go`, and add the build version (research R21):
  - Add `var version = "dev"` and `resolveVersion()`, which falls back to `debug.ReadBuildInfo().Main.Version` unless that is `(devel)`.
  - Add `-ldflags "-X main.version={{.Version}}"` to `.goreleaser.yaml` and `.goreleaser.darwin.yaml`.
  - Add `-ldflags "-X main.version=$(shell git describe --tags --always --dirty)"` to the `Makefile` `build` target.
  Then wire:
  1. `sqlite.NewEncounterStore(db)` → `app.NewService(repo)`
  2. `router := app.NewConfirmationRouter()`
  3. `conn := mcpserver.NewConnection(svc, router, mcpserver.Options{Version: resolveVersion(), Logf: logf})`
  4. `fyneui.Run(ctx, svc, cancel, fyneui.Options{Assistant: conn, Confirmations: router})`
- [x] T041 [P] Write `internal/ui/fyneui/assistant_connection_dialog_test.go` with `test.NewTempApp(t)` and a fake `AssistantConnection`:
  - toggling saves `enabled` and calls `Start`;
  - a `Start` error keeps the switch on and shows the reason;
  - the secret is masked by default;
  - the header label follows status changes.

**Checkpoint**: the app starts, the GM can turn the endpoint on, admission works, and the gate, events and revisions are in place. User stories can start.

---

## Phase 3: User Story 1 - Inspect Campaigns And Encounters From An Assistant (Priority: P1) 🎯 MVP

**Goal**: An assistant can read campaigns, rosters, encounters with difficulty, full encounter state, and operation logs.

**Independent Test**: Seed two campaigns and several encounters, connect a client, call every read tool, and compare each result with what the desktop app shows (spec US1 acceptance scenarios 1–6).

### Tests for User Story 1

- [x] T042 [P] [US1] Write `internal/mcpserver/tools_read_test.go`. It covers each US1 tool from `contracts/mcp-tools.md` against seeded data:
  - `list_campaigns` sets `is_active` correctly;
  - `get_active_campaign` on an empty DB returns `{initialized:false, campaign:null}` with no error;
  - `list_campaign_characters` includes `special`, `notes` and `available`, and defaults to the active campaign;
  - `list_encounters` difficulty equals `EncounterSummary` metrics;
  - `get_encounter` keeps initiative order and `is_current_turn` follows `TurnIndex`;
  - `get_active_encounter` returns null when there is none;
  - `get_encounter_log` is newest first, with `source`;
  - `list_party_members` and `list_monster_templates`;
  - an unknown `encounter_id` returns `not_found`;
  - every tool has `readOnlyHint:true, destructiveHint:false, idempotentHint:true, openWorldHint:false`;
  - no activity entry is written for reads;
  - Cyrillic names and multiline notes round-trip unchanged.
- [x] T043 [P] [US1] Write the performance test `internal/mcpserver/tools_read_perf_test.go`, skipped under `-short`: seed 50 campaigns, 200 encounters and 100 templates. Every read tool returns in under 1 s (SC-007).

### Implementation for User Story 1

- [x] T044 [US1] Create `internal/mcpserver/tools_read.go`:
  - **Tools**: `list_campaigns`, `get_active_campaign`, `list_campaign_characters` (optional `campaign_id`), `list_party_members`, `list_encounters`, `get_encounter`, `get_active_encounter`, `get_encounter_log` (`limit` 1..500, default 100), and `list_monster_templates`;
  - **Mapping**: through `dto.go`; `ErrCampaignNotInitialized` becomes `initialized:false` only for `get_active_campaign`;
  - **Descriptions**: each one explains the Fallout terms it returns (FR-020: Party AP max 6, GM Threat, DR per body location with poison global, immunity, torso-only, difficulty bands);
  - **Summary**: each result has a one-line human-readable summary in `content[0]`.
- [x] T045 [US1] Register the read tools in `internal/mcpserver/server.go` (a `registerReadTools(s, svc)` call).

**Checkpoint**: US1 is usable from Claude Code against a running tracker (quickstart §3 step 1, read-only questions).

---

## Phase 4: User Story 2 - Manage Campaigns Through The Assistant (Priority: P1)

**Goal**: Create a campaign, update its details, add a player, edit a character, change a character's availability, remove a player, and activate a campaign. All results match the CAMP screen. Removals are confirmed first.

**Independent Test**: Using only the assistant, create a 4-character campaign, edit one character's level, HP, S.P.E.C.I.A.L. and notes, mark another one unavailable (with confirmation), and activate the campaign. The desktop app shows exactly that (spec US2 acceptance scenarios 1–6).

### Tests for User Story 2

- [x] T046 [P] [US2] Write `internal/app/campaign_roster_patch_test.go`:
  - **Each patch kind**: details only; add player (a duplicate name, case-insensitive, is rejected); update character fields; rename returns the new `character_id` and the replaced old ID; mark available / mark unavailable (unavailable removes the character from every encounter); remove player.
  - **Atomicity**: everything runs under one write-gate hold (a concurrent desktop `ExecuteUpdateCampaign` cannot interleave).
  - **Sync**: linked active-encounter combatant sync still happens (spec 003 behavior).
- [x] T047 [P] [US2] Write `internal/mcpserver/confirm_test.go`:
  - **MRTR, client declares form elicitation on 2026-07-28**:
    - the first call returns `input_required` with `inputRequests.confirm` and a `requestState`;
    - a retry with accept and `confirm:true` runs the operation;
    - decline or cancel gives `declined` and no change;
    - an authentic `requestState` used after its 2-minute expiry returns error code `timed_out`, and the operation does not run;
    - a forged (bad HMAC) or reused `requestState`, or a retry with different arguments, returns `declined` with activity outcome `rejected`, and the operation does not run;
    - with an injectable clock, a token used once and replayed 30 s after its expiry returns `declined`/`rejected` (the nonce is still remembered), not `timed_out`;
    - `inputResponses` without any `requestState` start a fresh confirmation (`input_required` again);
    - the `input_required` step writes no activity entry, and the retry writes exactly one.
  - **Desktop fallback, client without elicitation**:
    - the fake confirmer answers accepted / declined / timed_out;
    - the confirmer receives a ctx whose deadline is 50 s or less;
    - `timed_out` surfaces to the client as error code `timed_out`, distinct from `declined`, and the operation does not run;
    - a client cancel makes the confirmer see a cancelled ctx, with outcome `declined`.
  - **Wording**: the confirmation text comes from stored state, never from tool arguments.
  - **No lock while waiting**: during a pending confirmation, a concurrent read tool completes.
- [x] T048 [P] [US2] Write `internal/mcpserver/tools_campaign_test.go`:
  - **Parity (SC-002)**: for each tool, the same change through `ExecuteUpdateCampaign` directly on a second fresh DB gives an identical `list_campaign_characters` result, ignoring IDs and timestamps.
  - **Validation**: invalid input (current HP above max HP, a S.P.E.C.I.A.L. value below 1, a negative DR) returns `invalid_input` with a message naming the field, and nothing is stored.
  - **Confirmation**: `remove_campaign_player` and `mark_character_unavailable` require it; `mark_character_available` does not.
  - **Annotations**: they match `contracts/mcp-tools.md`. In particular, `mark_character_unavailable` has `destructiveHint:true` and `mark_character_available` has `destructiveHint:false`.
  - **Activity**: entries carry outcome `done`, `rejected` or `declined`.

### Implementation for User Story 2

- [x] T049 [US2] Create `internal/app/campaign_roster_patch.go`:
  - `(*Service).ExecuteCampaignRosterPatch(ctx, CampaignRosterPatch)` with the kinds `Details{Name?, StartDate?}`, `AddPlayer`, `UpdateCharacter{CharacterID, fields…}`, `SetAvailability{CharacterID, Available}` and `RemovePlayer{CharacterID}`;
  - under one write-gate hold it reads `ListCampaignPlayers`, applies the patch, and calls the unexported implementation behind `ExecuteUpdateCampaign` (reusing `prepareCampaignPlayers`);
  - it returns the updated character and the `replaced_character_id` on rename;
  - it publishes a change event and bumps revisions per `data-model.md`.
- [x] T050 [US2] Create `internal/mcpserver/confirm.go`:
  - `confirmer.require(ctx, req *mcp.CallToolRequest, args any, cr app.ConfirmationRequest) (*mcp.CallToolResult, bool, error)`;
  - **MRTR path**: only when `req.ProtocolVersion() >= "2026-07-28"` and `req.ClientCapabilities().Elicitation` is set. `requestState` is an HMAC-SHA256 under a per-process random key over {tool, SHA-256 of canonical args JSON, expiry now+2m, nonce}, single-use via a set of used nonces. Each used nonce is kept until its token's expiry **plus 1 minute** before cleanup (research R5). The form schema is `{confirm: boolean}`.
    - **Verify in this order**: (1) HMAC; (2) arguments digest; (3) nonce unused. Any failure gives `declined`, activity `rejected`, detail "confirmation could not be verified". (4) If the token passes but is past its expiry, the result is `timed_out`. Only then is the action checked.
    - Only the final outcome is recorded in the activity log; the `input_required` step records nothing.
  - **Otherwise** `router.Confirm` with `context.WithTimeout(ctx, 50*time.Second)`. That stays below the 60 s client request timeout (research R5). A `timed_out` decision maps to error code `timed_out`;
  - never called while the gate is held.
- [x] T051 [US2] Create `internal/ui/fyneui/assistant_confirm_dialog.go`, `desktopConfirmer`, implementing `app.DesktopConfirmer`:
  - a FIFO queue showing one `dialog.NewCustomConfirm` at a time via `fyne.Do`;
  - title "Assistant wants to…", body = `ConfirmationRequest.Summary`;
  - Confirm (danger importance) / Cancel buttons;
  - a per-request 50-second deadline from enqueue time gives `timed_out`, and the dialog shows a seconds countdown and "N more requests waiting" when the queue is not empty;
  - a request whose deadline passes while queued is removed without being shown and returns `timed_out`;
  - ctx cancel closes the dialog, or removes the queued request, as `declined`;
  - `RequestFocus` on the window.
  Register it with `opts.Confirmations.SetDesktop(...)` in `internal/ui/fyneui/app.go`.
- [x] T052 [P] [US2] Write `internal/ui/fyneui/assistant_confirm_dialog_test.go`: FIFO order with two queued requests; the waiting counter shows "1 more request waiting"; timeout gives `timed_out` (injectable clock or short deadline); the second request expires while the first is still open, so it returns `timed_out` and its dialog is never shown; cancel of a queued request removes it as `declined`; Confirm gives `accepted`.
- [x] T053 [US2] Create `internal/mcpserver/tools_campaign.go`:
  - **Tools**: `create_campaign`, `update_campaign_details`, `add_campaign_player`, `update_campaign_character` (the description warns that a rename creates a new `character_id`), `mark_character_available`, `mark_character_unavailable` (destructive), `activate_campaign` and `remove_campaign_player`;
  - **Confirmation**: `remove_campaign_player` and `mark_character_unavailable` go through `confirm.require` with the stored-state wording from `contracts/mcp-tools.md`;
  - **Descriptions** (FR-020): explain the terms these tools use: DR per body location for physical, energy and radiation, global poison DR, immunity, S.P.E.C.I.A.L. minimum 1, current HP no greater than max HP, and that unavailable characters leave all encounters;
  - **Activity**: `recordActivity` for every outcome;
  - **Registration**: in `internal/mcpserver/server.go`.

**Checkpoint**: US1 and US2 both work. Campaign removals are confirmed in the Claude Code terminal or in the tracker window.

---

## Phase 5: User Story 3 - Prepare And Manage Encounters Through The Assistant (Priority: P1)

**Goal**: Preview difficulty for a draft roster; create, update, activate, restart and delete encounters; save monster templates. Restart, delete and roster removals are confirmed.

**Independent Test**: Through the assistant, create an encounter from templates and party members, compare its difficulty with the desktop calculator, change a monster quantity, then activate, restart and delete it (with confirmation). Verify each step in the app (spec US3 acceptance scenarios 1–6).

### Tests for User Story 3

- [x] T054 [P] [US3] Write `internal/app/encounter_draft_test.go`:
  - **Expansion**: `Quantity: 3` of a template gives 3 combatants with distinct generated IDs, the same as `collectCombatantsFromRows` (`internal/ui/fyneui/input_row_collectors.go:94-110`).
  - **Preview parity**: `PreviewEncounterDifficulty` equals `domain.EvaluateEncounterDifficulty` on the same expansion and stores nothing.
  - **Updates**: `KeepCombatantIDs` keep their stored HP and defeated state.
  - **Errors**: unknown template or party character returns `not_found`; quantity outside 1..50 or a `DraftMonster` with both or neither of `template_id`/`monster` returns `invalid_input`; a roster with no combatants is rejected.
- [x] T055 [P] [US3] Write `internal/mcpserver/tools_encounter_test.go`:
  - **Parity (SC-002)**: `create_encounter` against `ExecuteCreateEncounter` with the same expanded combatants.
  - **Confirmation**: `update_encounter` with every stored combatant listed in `keep_combatant_ids` (rename or additions only) runs without confirmation; dropping one requires it; `restart_encounter` and `delete_encounter` always require it, and a declined confirmation leaves the encounter unchanged.
  - **Preserved behavior**: `update_encounter` on a non-active encounter makes it active (research R20); `restart_encounter` resets only NPC HP and defeated state; deleted encounters are not listed and return `not_found` afterwards.
  - **Templates**: `save_monster_template` reports `replaced_existing` on a case-insensitive name match.
  - **Log source**: `activate_encounter` writes "Encounter activated" with `source=assistant`.
  - **Annotations and activity**: annotations match the contract, and activity outcomes are recorded.

### Implementation for User Story 3

- [x] T056 [US3] Create `internal/app/encounter_draft.go`:
  - `EncounterDraft{Name; PartyCharacterIDs []string; Monsters []DraftMonster; KeepCombatantIDs []string}` and `DraftMonster{TemplateID string; Monster *domain.Combatant; Quantity int; Initiative *int}`;
  - `expandDraft`, unexported and running under the caller's gate hold, resolves party members through the repository party-member list and templates through the repository template list;
  - `(*Service).PreviewEncounterDifficulty(ctx, EncounterDraft) (domain.EncounterDifficultyMetrics, error)` takes the read gate;
  - `(*Service).ExecuteCreateEncounterFromDraft(ctx, EncounterDraft)` and `(*Service).ExecuteUpdateEncounterFromDraft(ctx, encounterID string, EncounterDraft)` take the write gate and reuse the unexported bodies of `ExecuteCreateEncounter` / `ExecuteUpdateEncounter` (including `prepareEncounterCombatants` and `resolveLinkedPartyCombatants`);
  - every path publishes events and bumps revisions.
- [x] T057 [US3] Create `internal/mcpserver/tools_encounter.go`:
  - **Tools**: `preview_encounter_difficulty` (read-only annotations), `create_encounter`, `activate_encounter`, `save_monster_template`, `update_encounter`, `restart_encounter` and `delete_encounter`;
  - **Confirmation**: `update_encounter` needs it only when stored combatants are missing from `keep_combatant_ids`; `restart_encounter` and `delete_encounter` always. The wording comes from stored state, as in `contracts/mcp-tools.md`;
  - **Descriptions** (FR-020): `update_encounter` says "Saving an encounter makes it the active encounter (tracker behavior)". `preview_encounter_difficulty` and `create_encounter` explain the difficulty formula and bands. `MonsterInput` fields explain torso-only creatures and per-location DR. `restart_encounter` states that only NPC HP and defeated state reset;
  - **Registration**: in `internal/mcpserver/server.go`.

**Checkpoint**: US1–US3 work. Encounter preparation through the assistant is complete.

---

## Phase 6: User Story 6 - Control The Connection From The Desktop App And See Changes Live (Priority: P1)

**Goal**: Full connection control (port, secret regeneration, setup snippets), the activity log in the UI and for the assistant, live refresh of desktop views, stale-edit protection in editors, and the `[AI]` marker in DATA.

**Independent Test**: With the STAT screen open, have the assistant edit a character, create and activate an encounter, and apply damage. Every change appears within 1 s with no manual refresh. An open editor warns before saving over an assistant change. The activity list shows all actions and refusals (spec US6 acceptance scenarios 1–9, quickstart §§2 and 5).

### Tests for User Story 6

- [x] T058 [P] [US6] Write `internal/ui/fyneui/assistant_refresh_test.go`:
  - assistant events within 100 ms coalesce into exactly one `Refresh` (fake timer);
  - desktop-sourced events trigger nothing;
  - an open editor's entry text survives a refresh.
- [x] T059 [P] [US6] Write `internal/ui/fyneui/stale_edit_test.go` (fake service responses):
  - **Encounter editor**: `ErrStaleEdit` shows Reload/Overwrite and saves nothing until a choice is made; Overwrite resubmits with the current revision; Reload reopens with current data; `ErrEntityDeleted` shows a close-only notice and does not recreate; `ErrTrackerBusy` keeps the editor open with the input intact.
  - **Campaign editor**: the same `ErrStaleEdit` handling.
- [x] T060 [P] [US6] Write `internal/ui/fyneui/encounter_list_dialog_test.go` (extend the existing one if present): a reload keeps the selection; deleting the selected encounter clears the selection and shows a note.
- [x] T061 [P] [US6] Write `internal/mcpserver/activity_test.go` (SC-010):
  - each modifying or destructive operation writes exactly one entry, for its final outcome, with the correct outcome, entity kind, ID and label. With in-conversation confirmation, the `input_required` call writes none and the retry writes one. An abandoned confirmation (no retry) writes none;
  - read tools write none;
  - declined and timed-out confirmations are recorded;
  - `list_assistant_activity` returns entries newest first, respects `limit` 1..200, and offers no write path;
  - the secret never appears in any entry.
- [x] T062 [P] [US6] Write `internal/mcpserver/concurrency_test.go`, run with `-race`: assistant `apply_damage`/`advance_turn` tool calls run in parallel with direct `svc.ExecuteApplyDamage`/`AdvanceTurn` calls that simulate the UI. No lost updates and no busy errors under a normal load (US6 acceptance scenario 4).
- [x] T063 [P] [US6] Write `internal/ui/fyneui/data_log_test.go` (extend if present): an assistant entry renders as `[<ts>] [R<n>] [AI] <message>`, and desktop entries stay unchanged.

### Implementation for User Story 6

- [x] T064 [US6] Add `list_assistant_activity` (`limit` 1..200, default 20, read-only annotations) to `internal/mcpserver/tools_read.go`, backed by `svc.ListAssistantActivity`.
- [x] T065 [US6] Create `internal/ui/fyneui/assistant_refresh.go`:
  - subscribe via `svc.SubscribeChanges` in `Run`, and ignore `Source == desktop`;
  - coalesce over 100 ms with `time.AfterFunc`;
  - then `fyne.Do(func(){ refresher.Refresh(); openEncounterList.reload(); openAssistantDialog.reloadActivity() })`;
  - unsubscribe on shutdown.
- [x] T066 [US6] In `internal/ui/fyneui/encounter_list_dialog.go`, expose a `reload()` that keeps the selection by encounter ID and clears it with an inline note when the encounter is gone. Register and unregister the open dialog with the refresher from T065.
- [x] T067 [US6] In `internal/ui/fyneui/encounter_editor_dialog.go`, capture `svc.Revision({encounter, id})` when the edit dialog opens and pass `ExpectedRevision` on save. Handle `ErrStaleEdit`, `ErrEntityDeleted` and `ErrTrackerBusy` exactly as in the `contracts/desktop-ui.md` "Stale-edit handling" table.
- [x] T068 [US6] Do the same for `internal/ui/fyneui/campaign_dialogs.go`, the campaign edit path (`UpdateCampaign` at `:276`/`:291`): capture the revision and handle `ErrStaleEdit`/`ErrTrackerBusy` per `contracts/desktop-ui.md`.
- [x] T069 [US6] Render `[AI]` for `Source == assistant` in `internal/ui/fyneui/data_log.go` (`refreshEncounterDataLog`).
- [x] T070 [US6] Complete `internal/ui/fyneui/assistant_connection_dialog.go`:
  - a port entry and Apply: accept 1024–65535 with a field-level message otherwise, then `conn.Apply`;
  - Regenerate secret: confirm, then `svc.RegenerateAccessSecret` + `conn.Apply`, with a note that clients must be updated;
  - setup snippets for Claude Code and Claude Desktop, filled with the real port, secret and `os.Executable()` path, each with Copy;
  - an activity list (newest 200, newest first: time · operation · entity · outcome · detail) with Clear (confirm, then `svc.ClearAssistantActivity`).
- [x] T071 [P] [US6] Extend `internal/ui/fyneui/assistant_connection_dialog_test.go`:
  - an invalid port shows a field message and does not call `Apply`;
  - regenerate calls `Apply` with the new secret;
  - the snippets contain the current port and not a stale secret;
  - Clear asks for confirmation first.

**Checkpoint**: all P1 stories are done. The tracker and the assistant stay consistent during play.

---

## Phase 7: User Story 4 - Run Live Combat Through The Assistant (Priority: P2)

**Goal**: Advance the turn, apply typed and located damage, heal, and change Party AP and GM Threat on the active encounter, with the outcome reported.

**Independent Test**: Activate an encounter, then through the assistant advance the turn twice, apply damage of each type to a combatant with DR and immunity, heal a party member, and change AP and Threat. The results and stored state match the same desktop actions (spec US4 acceptance scenarios 1–7).

### Tests for User Story 4

- [x] T072 [P] [US4] Write `internal/mcpserver/tools_combat_test.go`:
  - **`advance_turn`**: returns the new `current_combatant` and round.
  - **`apply_damage`**:
    - all four damage types × the locations against a combatant with per-location DR;
    - an immune type gives `applied:0`;
    - torso-only creatures always use the torso;
    - poison uses the global DR;
    - HP clamps at 0 and sets `defeated:true`;
    - an unknown `combatant_id` returns `not_found`;
    - an invalid enum returns `invalid_input`.
  - **`heal`**: caps at max HP and reports `healed`.
  - **AP and Threat**:
    - `change_party_ap(+5)` from 3 gives `{party_ap:6, capped:true}`;
    - overspending AP or Threat returns `invalid_input` and leaves resources unchanged;
    - `delta:0` returns `invalid_input`.
  - **No active encounter**: every combat tool returns `no_active_encounter`.
  - **Parity (SC-002)**: results match direct `ExecuteApplyDamage`/`ExecuteHeal`/`AdvanceTurn`/`AddPartyAP`/`SpendThreat`.
  - **Log source and correction**: operation-log entries carry `source=assistant`; a correction found through `list_assistant_activity` and applied with the opposite `heal` restores HP (US4 acceptance scenario 7).

### Implementation for User Story 4

- [x] T073 [US4] Create `internal/mcpserver/tools_combat.go`:
  - **Tools**: `advance_turn`, `apply_damage` (enums `physical|energy|radiation|poison` and `head|torso|left_arm|right_arm|left_leg|right_leg`, `amount >= 0`), `heal` (`amount >= 0`), `change_party_ap` and `change_gm_threat` (`delta != 0`: positive → `AddPartyAP`/`AddThreat`, negative → `SpendPartyAP`/`SpendThreat`);
  - **Results**: per the `contracts/mcp-tools.md` modifying-tools table;
  - **Descriptions**: explain the DR, immunity, torso-only, poison, AP cap and Threat rules (FR-020);
  - **Activity**: record it, with detail such as "7 physical → torso, applied 4";
  - **Registration**: in `internal/mcpserver/server.go`.

**Checkpoint**: live combat through the assistant works and is visible live in the tracker (thanks to US6).

---

## Phase 8: User Story 5 - Guided GM Workflows Shipped With The Tracker (Priority: P3)

**Goal**: Three SEP-2640 skills plus a rules reference, delivered with the binary. They are available as skills to clients that support them, and as resources and instructions to the rest.

**Independent Test**: In Claude Code 2.1.332 or later, `/fallout-tracker:prepare-encounter a hard fight` loads the skill, iterates `preview_encounter_difficulty`, and saves an encounter labelled Hard. A client without skills support lists and reads the same `skill://` resources (spec US5 acceptance scenarios 1–3, SC-009).

### Tests for User Story 5

- [x] T074 [P] [US5] Write `internal/mcpserver/skills_test.go`:
  - **Extension**: `server/discover` capabilities include `extensions["io.modelcontextprotocol/skills"]` with `directoryRead:false` alongside `resources`.
  - **`skills/list`**: returns 3 skills; each `resources[0].digest == "sha256:"+hex(sha256(resources/read bytes))` and `size` matches; results carry `resultType:"complete"`, `ttlMs` and `cacheScope`.
  - **`skills/get`**: an unknown URI returns error -32602.
  - **Frontmatter**: each `name` matches `^[a-z0-9]+(-[a-z0-9]+)*$`, is at most 64 characters, and equals its directory; `description` is 1–1,024 characters; `metadata.version` equals the build version; each body is under 500 lines.
  - **Fallback**: `resources/list` includes the three `skill://…/SKILL.md` URIs and `fallout://reference/rules` for a client without the extension.
  - **Instructions**: under 1,500 characters, mentioning the 6-point AP cap, confirmation, and "saving an encounter makes it active".

### Implementation for User Story 5

- [x] T075 [P] [US5] Write `internal/mcpserver/skills/start-campaign/SKILL.md`. Frontmatter: `name: start-campaign`, `description` (when to use it: creating a campaign and a party), `metadata.version`. The body is step-by-step guidance:
  - gather the players and characters;
  - required fields and defaults (S.P.E.C.I.A.L. = 1, DR per location, poison global);
  - call `create_campaign`, then `activate_campaign`;
  - verify with `list_campaign_characters`.
- [x] T076 [P] [US5] Write `internal/mcpserver/skills/prepare-encounter/SKILL.md`. Frontmatter as in T075, `name: prepare-encounter`. The body:
  - read `list_party_members` and `list_monster_templates`;
  - the difficulty formula and the bands (Trivial < −2, Simple −2..−1, Average 0..1, Hard 2..5, Deadly > 5);
  - iterate `preview_encounter_difficulty` by adjusting quantities, then `create_encounter`;
  - report the label;
  - create templates for new monsters with `save_monster_template`.
- [x] T077 [P] [US5] Write `internal/mcpserver/skills/run-combat-round/SKILL.md`. Frontmatter as in T075, `name: run-combat-round`. The body:
  - `get_active_encounter` and turn order, then `advance_turn`;
  - `apply_damage` with the type and location rules (torso-only, poison global, immunity) and `heal`;
  - AP (cap 6) and Threat spend and gain;
  - defeated combatants;
  - correcting mistakes with `list_assistant_activity` and opposite operations (there is no undo);
  - restarting or deleting asks for the GM's confirmation.
- [x] T078 [P] [US5] Write `internal/mcpserver/reference/rules.md` (served as `fallout://reference/rules`): the difficulty formula and bands, the Party AP cap, the GM Threat rules, the DR model by damage type and location, immunity, and torso-only creatures. Mirror `README.md` "Encounter Difficulty" and `internal/domain` behavior.
- [x] T079 [US5] Create `internal/mcpserver/skills.go`:
  - `//go:embed skills reference`;
  - at startup, compute the sha256 digest and size of each file and inject `metadata.version` from the build version into the served frontmatter (digests are computed **after** injection);
  - register `skills/list` and `skills/get` with `mcp.AddReceivingCustomMethod` (results embed `ResultBase`, with `resultType`, `ttlMs` and `cacheScope`);
  - advertise `Capabilities.AddExtension("io.modelcontextprotocol/skills", {"directoryRead": false})`;
  - register the `skill://<name>/SKILL.md` and `fallout://reference/rules` resources (`text/markdown`) with `AddResource`;
  - keep everything behind a `skillProvider` interface so the code can later switch to the go-sdk `extensions/skills` package.
- [x] T080 [US5] Write the final `Instructions` text in `internal/mcpserver/server.go`, under 1,500 characters: the three guides by URI, the rules reference URI, "destructive tools ask the GM to confirm", "the Party AP cap is 6", and "saving an encounter makes it the active encounter". Call the skills registration from T079.

**Checkpoint**: every user story is functional.

---

## Phase 9: Claude Desktop Bridge (Cross-Cutting, FR-008, SC-006, SC-011)

**Purpose**: Claude Desktop Chat can reach the embedded endpoint through `fallout-tracker mcp-bridge`.

- [x] T081 [P] Write `internal/mcpserver/bridge_test.go`:
  - **Relay**: a go-sdk client on an `io.Pipe` stdio transport talks to `runBridge` (in-process), which talks to the httptest endpoint from T035. `initialize` / `tools/call list_campaigns` succeed for a 2025-11-25 client and for a 2026-07-28 client; notifications produce no output.
  - **Headers**: `Mcp-Method` and `Mcp-Name` are derived (assert on a recording handler), and `Authorization` is forwarded.
  - **Unreachable tracker**: every request gets JSON-RPC `-32000` with the "not running" text.
  - **401**: gives `-32001`.
  - **Timeout**:
    - the bridge's HTTP client timeout is a `bridgeOptions.HTTPTimeout` field, and its default equals 150 s (assert the default value);
    - with `HTTPTimeout` set to 300 ms, a handler that blocks 200 ms returns its result through the bridge, and a handler that blocks 400 ms gets the JSON-RPC error `-32000`;
    - no test waits in real time for more than a second.
  - **Exit code**: 2 with a missing `FALLOUT_TRACKER_MCP_SECRET`.
- [x] T082 Create `internal/mcpserver/bridge.go`:
  - `RunBridge(ctx, stdin io.Reader, stdout, stderr io.Writer, env func(string) string) int`;
  - newline-delimited JSON-RPC relay to `FALLOUT_TRACKER_MCP_URL` (default `http://127.0.0.1:47823/mcp`);
  - headers per the `contracts/mcp-connection.md` bridge table, with the negotiated `MCP-Protocol-Version` taken from the initialize response;
  - the HTTP client timeout comes from `bridgeOptions.HTTPTimeout` (default 150 s, never shorter in production), so desktop confirmations (50 s) pass through. `RunBridge` uses the default; tests inject a short value;
  - no database and no Fyne.
- [x] T083 In `cmd/fallout-tracker/main.go`, dispatch `os.Args[1] == "mcp-bridge"` to `os.Exit(mcpserver.RunBridge(ctx, os.Stdin, os.Stdout, os.Stderr, os.Getenv))` **before** resolving the DB path or creating any Fyne objects.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [x] T084 [P] Write the GM setup guide `docs/assistant.md`:
  - enable the connection;
  - Claude Code `claude mcp add --scope user --transport http … --header "Authorization: Bearer …"`;
  - Claude Desktop `claude_desktop_config.json` with `mcp-bridge` and `FALLOUT_TRACKER_MCP_SECRET`, plus the macOS/Linux/Windows executable paths;
  - the `mcp-remote` alternative;
  - what each client supports (skills and elicitation in Claude Code; resources and the desktop confirmation dialog in Desktop Chat);
  - rotating the secret;
  - troubleshooting: port in use, 401, "tracker is not running", and skills not showing (Claude Code version, plan admin setting).
- [x] T085 [P] Add a short "Assistant (MCP)" section to `README.md` that links `docs/assistant.md` and lists the safety model (off by default, loopback only, access secret, confirmations, activity log).
- [x] T086 Run the repository skills `.claude/skills/go-code-quality-review/SKILL.md` and `.claude/skills/code-simplification-review/SKILL.md` over all changed Go files (`internal/app`, `internal/mcpserver`, `internal/store/sqlite`, `internal/ui/fyneui`, `cmd/fallout-tracker`) and apply the findings without changing behavior.
- [x] T087 Run the quality gates and record the output in the implementation report:
  1. `make sqlc-generate`
  2. `make db-doc-generate`
  3. `git diff --exit-code internal/store/sqlite/dbgen docs/db`
  4. `go test ./...`
  5. `go test -race ./internal/app/... ./internal/mcpserver/...`
  6. `go vet ./...`
  7. `make lint`
  8. `make build`
- [x] T088 Run the manual checks in `specs/004-mcp-server/quickstart.md` §§2–5 in **both** Claude Code and Claude Desktop (SC-001, SC-004, SC-006, SC-009, SC-011).
  - Run the §4a trial protocol: SC-004 needs at least 19 of 20 trials within 1 s; SC-009 needs at least 4 of 5 trials labelled `Hard`.
  - Record how many seconds Claude Desktop waits before it cuts a pending desktop confirmation (quickstart §4 step 2). If that is 50 s or less, reopen research R5.
  - Record pass/fail per step and per trial, the timings, and the client versions in the "Validation log" section of `specs/004-mcp-server/quickstart.md`.
- [x] T089 Classify tool errors by error kind, not by message wording (BUG-001, FR-025, research R22):
  - **Domain** (`internal/domain/errors.go`): add `ErrInvalidInput`, `ErrNotFound`, `InvalidInput(err)` and `NotFound(err)`. They keep the message and are idempotent. Make `ErrEncounterNotFound` and `ErrCampaignNotFound` wrap `ErrNotFound`. Mark unknown combatants in `internal/domain/encounter.go` as not found.
  - **Application** (`internal/app/errors.go`): add `invalidf` and `notFoundf`. Mark the checks in every Service file, the `prepare*` validation helpers, and the results of `ApplyDamage`, `Heal`, `AdvanceTurn`, resource spends, settings validation and the activity limit.
  - **Store** (`internal/store/sqlite/{campaign,encounter,monster_template,encounter_logs}_repository.go`): mark user-caused "required" guards and the "already" duplicates as invalid input.
  - **Adapter** (`internal/mcpserver/errors.go`): map by `errors.Is` only and remove the word list; anything unmarked is `internal`.
  - **Tests**: `internal/domain/errors_test.go`; `internal/mcpserver/errors_test.go` (unmarked validation-like text gives `internal`); the guard test `internal/app/errors_test.go` (every user-caused Service error is marked).
  - **Docs**: update the `contracts/mcp-tools.md` error table and research R22. Then run `go test ./...`, `go test -race ./internal/app/... ./internal/mcpserver/...`, `go vet ./...` and `make lint`.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: T001 → T002 → T004. T003 runs in parallel with T002. If T004 fails, including the manual header capture, stop and re-plan. T032 must not start before T004's header capture is recorded in research R3.
- **Foundational (Phase 2)**: depends on Setup and blocks every story.
  - The domain tasks (T005–T008) come first.
  - Persistence: T009 → T010 → T011 → T013/T014 → T015/T016 → T017.
  - App core: T018 → T019–T023 → T024/T025. T026 → T027 → T028.
  - MCP core (T029–T037) needs T026 and T027.
  - Desktop wiring (T038–T041) needs T034.
- **US1 (Phase 3)**: after Foundational.
- **US2 (Phase 4)**: after Foundational. It introduces `confirm.go` and the desktop confirmer, which US3 reuses.
- **US3 (Phase 5)**: after US2's T050–T052 (confirmation). Otherwise independent of US2's tools.
- **US6 (Phase 6)**: after Foundational. Its full manual test uses US2 and US3 tools. T064 needs T044.
- **US4 (Phase 7)**: after Foundational. Its live-refresh check uses US6 T065. Its correction test in T072 (US4 acceptance scenario 7) needs `list_assistant_activity` from US6 T064.
- **US5 (Phase 8)**: after Foundational. Its playbooks reference tools from US1–US4, so finish their tools first for SC-009.
- **Bridge (Phase 9)**: after Foundational. It is best run after US1 so there are real tools to relay.
- **Polish (Phase 10)**: after all the desired stories. T089 (BUG-001) follows T029 and the tool phases whose errors it classifies (T044, T053, T057, T073).

### Story dependency graph

```text
Setup ─► Foundational ─┬─► US1 ──────────────┐
                       ├─► US2 ─► US3 ───────┤
                       ├─► US6 (uses US2/3) ─┼─► US5 ─► Bridge ─► Polish
                       └─► US4 (uses US6: T064, T065) ───┘
```

### Within each story

- Tests first, and they must fail. Then the app layer, then the adapter tools, then registration and UI.
- Every new mutating `Service` method must take the write gate, publish a change event and bump revisions, per T021–T023.

## Parallel Opportunities

- **Phase 1**: T003 runs in parallel with T002.
- **Phase 2**:
  - T005, T006, T007 and T012 can run together.
  - T015 and T016 can run together after T014.
  - T019 runs in parallel once T018 is done.
  - T024, T025 and T028 can run together.
  - T029, T030, T035, T036 and T037 can run together after T033 and T034 exist.
- **US1**: T042 and T043 can run together.
- **US2**: T046, T047, T048 and T052 can run together. T050 and T051 touch different packages and can run together.
- **US3**: T054 and T055 can run together.
- **US6**: all of T058–T063 can run together. T066–T069 touch different files and can run together after T065.
- **US5**: T075–T078 are content files and can run together. T074 runs in parallel with them.
- **Phase 10**: T084 and T085 can run together.

## Parallel Example: User Story 2

```bash
# Tests first (all [P]):
Task: "Write internal/app/campaign_roster_patch_test.go (patch kinds, single gate hold)"
Task: "Write internal/mcpserver/confirm_test.go (MRTR + desktop fallback + forged/reused state)"
Task: "Write internal/mcpserver/tools_campaign_test.go (parity, validation, confirmations, activity)"
Task: "Write internal/ui/fyneui/assistant_confirm_dialog_test.go (FIFO, timeout, cancel)"

# Then implementation in different packages in parallel:
Task: "Create internal/mcpserver/confirm.go"
Task: "Create internal/ui/fyneui/assistant_confirm_dialog.go"
```

## Implementation Strategy

### MVP first (US1 on top of Foundational)

1. Phase 1, including the T004 spike, then Phase 2. The GM can turn on a secured endpoint.
2. Phase 3 (US1). A read-only assistant answers questions about campaigns and encounters.
3. **Stop and validate**: `go test ./internal/mcpserver/...` and quickstart §§2–3 step 1 in Claude Code.

### Incremental delivery

1. US2, then US3 (campaign and encounter management with confirmations). Validate quickstart §3 steps 2–5.
2. US6: full desktop control, live refresh and stale-edit protection. Validate quickstart §§2 and 5.
3. US4: live combat. Validate quickstart §3 steps 4 and 6.
4. US5: skills and reference content. Validate quickstart §3 step 3.
5. Bridge, then Polish: Claude Desktop, docs and gates. Validate quickstart §4 and run T087 and T088.

## Notes

- `[P]` means different files and no dependency on unfinished tasks.
- Never hand-edit `internal/store/sqlite/dbgen/`. Regenerate it with `make sqlc-generate` (constitution Principle V).
- The adapter (`internal/mcpserver`) validates only the wire format. Business rules stay in `internal/domain` and `internal/app` (Principle II).
- The access secret must never appear in logs, errors, activity entries or test failure output.
- Commit after each task or logical group, using Conventional Commit prefixes (`feat:`, `test:`, `docs:`, `chore:`).

**Bugfix**: 2026-11-27 — BUG-001 Updated from bugfix patch: added T089 (error kinds instead of message wording).

## Phase 11: Convergence

- [x] T090 Make `Connection.Apply` in `internal/mcpserver/connection.go` start the endpoint on the new port when the connection is enabled but its last start failed. Choosing another port and pressing APPLY must recover from a busy port, with a test in `internal/mcpserver/connection_test.go`, per FR-006 and US6/AC7 (partial)
- [x] T091 Make `Connection.Stop` in `internal/mcpserver/connection.go` set the status to `stopped` and notify listeners even when nothing is listening, so turning the switch off after a failed start clears "Failed" and the header shows `ASSISTANT · OFF`. Add a test in `internal/mcpserver/connection_test.go`, per FR-004 (partial)
- [x] T092 Cancel pending confirmations when the connection stops:
  - in `internal/mcpserver/connection.go`, derive confirmation contexts from a connection-lifetime context that `Stop` cancels;
  - in `internal/mcpserver/confirm.go`, use that context, so a desktop confirmation still open at shutdown closes as `declined` and its operation never runs;
  - in `internal/ui/fyneui/assistant_connection_dialog.go`, run `Stop` off the UI goroutine so it cannot freeze the UI;
  - add tests,

  per desktop-ui "Shutdown", FR-007 and R16 (contradicts)
- [x] T093 Make `internal/mcpserver/bridge.go` cancel the in-flight HTTP request whose id matches a `notifications/cancelled` from the client, so a cancel through the bridge closes a pending desktop confirmation as `declined`. Add a test in `internal/mcpserver/bridge_test.go`, per contracts/mcp-tools.md "Confirmation flow" step 2 and FR-008 (partial)
- [x] T094 Re-read the campaign by ID in `openCampaignEditor` (`internal/ui/fyneui/campaign_dialogs.go`) before capturing its revision, so the editor never opens with a stale name or start date from the campaign list and silently reverts an assistant change on save. Add a test, per FR-022 and US6/AC5 (partial)
- [x] T095 Add JSON Schema constraints to tool inputs in `internal/mcpserver/server.go` and the input types, as contracts/mcp-tools.md "Inputs" requires:
  - `minimum`/`maximum` for amount, quantity 1..50, limit ranges, level, hp, max_hp, defense, DR and S.P.E.C.I.A.L.;
  - `minLength: 1` for identifiers;
  - enums for `side` and `source`;
  - a schema test in `internal/mcpserver/tools_read_test.go`,

  per contract: mcp-tools Inputs and T030 (missing)
- [x] T096 Move the case-insensitive duplicate-player-name rule from `internal/mcpserver/tools_campaign.go` into `internal/app` (`prepareCampaignPlayers` for create and update, Unicode-aware with `strings.ToLower`), marked `domain.ErrInvalidInput`. The desktop and the assistant must reject the same rosters (Ann/ann, Иван/иван), with app and adapter tests, per FR-009, SC-002, FR-025 and Constitution II (contradicts)
- [x] T097 Capture `svc.Revision` before reading the encounter in `openEncounterEditor` (`internal/ui/fyneui/encounter_list_dialog.go`), so an assistant write between the read and the capture still triggers the stale-edit warning, per FR-022 (partial)
- [x] T098 Record the refusals collapsed in the final window in `internal/mcpserver/admission.go`, for example by flushing the count when the window ends or on `Stop`, so `detail` counts every suppressed refusal. Add a test in `internal/mcpserver/admission_test.go`, per R13 and data-model "Refused requests" (partial)
- [x] T099 In `internal/mcpserver/activity.go`, store the generic internal-error message (not the raw `err.Error()`) as the activity detail for errors that map to `internal`, so `list_assistant_activity` never exposes internal or SQL text. Add a test in `internal/mcpserver/activity_test.go`, per contracts/mcp-tools.md error table `internal` (contradicts)
- [x] T100 Make `remove_campaign_player` in `internal/app/campaign_roster_patch.go` (`RemovePlayer`) atomic: one save and one change event, or a rollback of the first save when the second fails. Add a test, per FR-017 and T025 (partial)
- [x] T101 Return `capped` and `replaced_existing` from `internal/app` within one gate hold (AP addition in `internal/app/encounter_service.go`, template save in `internal/app/monster_template_service.go`) instead of re-deriving them in `internal/mcpserver/tools_combat.go` and `internal/mcpserver/tools_encounter.go`, per FR-009 and Constitution II (contradicts)
- [x] T102 Record in `specs/004-mcp-server/research.md` R3 that admission uses its own constant-time bearer check rather than `auth.RequireBearerToken`, and why (missing-token refusals must be logged), per plan: R3 / T032 (contradicts)
- [x] T103 Align the bridge headers table in `specs/004-mcp-server/contracts/mcp-connection.md` with R8 and `internal/mcpserver/bridge.go`: no `MCP-Protocol-Version` on `initialize`, and `Mcp-Method`/`Mcp-Name` only for protocol 2026-07-28 or later, per contract: mcp-connection bridge vs plan: R8 (contradicts)
- [x] T104 Name `gopkg.in/yaml.v3` (skill frontmatter) and `github.com/google/jsonschema-go` (tool schemas) as direct dependencies in `specs/004-mcp-server/plan.md` Technical Context, or remove them, per plan: Technical Context dependencies (unrequested)
- [x] T105 Document in `specs/004-mcp-server/contracts/mcp-tools.md` that `update_encounter` re-checks the removal set after confirmation and returns `invalid_input` on `keep_combatant_ids` when the roster changed meanwhile (`internal/mcpserver/tools_encounter.go` `sameRemovals`), per FR-015 and SC-003 (unrequested)
