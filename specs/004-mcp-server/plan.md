# Implementation Plan: Assistant Control Of Campaigns And Encounters (MCP Server)

**Branch**: `004-mcp-server` | **Date**: 2026-11-27 | **Spec**: `specs/004-mcp-server/spec.md`

**Input**: Feature specification from `/specs/004-mcp-server/spec.md`

## Summary

The tracker gets an MCP server embedded in the desktop process, so Claude Code and Claude Desktop can inspect and manage campaigns and encounters and run live combat, using exactly the same application rules as the Fyne UI.

### Connection and security

- The server uses go-sdk v1.8.0 over a **stateless Streamable HTTP endpoint**. Statelessness is required for MCP 2026-07-28, which Claude Code skills and MRTR elicitation depend on.
- The endpoint is bound to `127.0.0.1` and stays off until the GM turns it on.
- Every request must present a bearer secret. Browser requests are refused, and the `Host` header is checked against DNS rebinding.
- Claude Desktop Chat reaches the endpoint through a built-in stdio relay, `fallout-tracker mcp-bridge`.

### Tools and confirmations

- The server exposes 30 tools: read, modify, and destructive.
- Destructive tools ask the GM to confirm. Clients that support form elicitation use MRTR, with an HMAC-protected, single-use `requestState`. Other clients fall back to a dialog in the tracker window.
- The confirmation wait happens outside the operation lock.

### Skills and reference content

- Three GM playbooks ship as SEP-2640 skills, hand-rolled because no released Go SDK has the extension yet.
- The same files are also served as ordinary resources, together with server instructions, for clients without skills.

### Changes in `internal/app`

- **Operation gate:** a context-aware reader/writer gate that serializes UI and assistant writes.
- **Operation source:** every request carries its source in the context. It tags encounter log entries and change events.
- **Change events:** after each write, an event drives the live Fyne refresh through `fyne.Do`.
- **Stale-edit detection:** in-memory entity revisions let open desktop editors detect changes made underneath them and offer Reload or Overwrite.
- **Draft rosters:** a shared draft-roster expansion is used by both the difficulty preview and encounter saves.

### Storage, UI, and governance

- Migration `00044` adds connection settings, the assistant activity log, and `encounter_logs.source`.
- The UI gains an ASSISTANT dialog and a confirmation dialog.
- The constitution is amended to v1.2.0 to name the new adapter and the loopback-only network exception.

## Technical Context

**Language/Version**: Go 1.27.4

**Primary Dependencies**:

- Existing: Fyne v2.7.4, modernc.org/sqlite v1.50.1, Goose v3.27.1, sqlc (tools module), google/uuid, testify.
- **New:** `github.com/modelcontextprotocol/go-sdk` v1.8.0, which brings `google/jsonschema-go` and `yosida95/uritemplate` transitively.
- **Promoted from indirect to direct:** `golang.org/x/sync` v0.20.0 (`semaphore`), `github.com/google/jsonschema-go` (tool input and output schemas, `confirm.go` form schema), and `gopkg.in/yaml.v3` (`SKILL.md` frontmatter in `skills.go`).

**Storage**:

- The local SQLite tracker.db.
- Migration `00044_add_assistant_connection.sql` adds two tables (`assistant_connection_settings`, `assistant_activity`) and the column `encounter_logs.source`.
- The DSN gains `busy_timeout(5000)`.

**Testing**:

- `go test` with testify.
- MCP integration tests use `httptest` with the go-sdk client (`StreamableClientTransport`) against temp-dir SQLite services.
- Bridge tests pipe a stdio client through the relay.
- Fyne tests use `fyne.io/fyne/v2/test`.
- `go test -race` runs for `internal/app` and `internal/mcpserver`.

**Target Platform**: Native desktop on macOS, Linux, and Windows. The listener uses loopback TCP only.

**Project Type**: A single local desktop application with layered internals and a new second inbound adapter (MCP) next to the Fyne UI.

**Performance Goals**:

- Read tools respond in under 1 s at 50 campaigns, 200 encounters, and 100 templates (SC-007).
- The desktop shows assistant changes within 1 s (SC-004), with 100 ms coalescing.
- Gate contention is bounded by the 5 s operation timeout and then returns `busy`.

**Constraints**:

- Offline. There is no remote access.
- The endpoint is off by default and protected by a secret.
- No operation lock is held during a confirmation: the desktop dialog waits up to 50 s, and an elicitation token lives 2 minutes.
- No combat rule is reimplemented in an adapter.
- The secret never appears in logs, activity entries, or errors.
- Generated code (`dbgen`) is regenerated, never edited by hand.

**Scale/Scope**:

- One GM and one desktop process. Claude Code and Claude Desktop may be connected at the same time.
- 30 tools, 4 resources, and 3 skills.
- Activity log capped at 1,000 entries.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Layer boundaries | **Pass with amendment** | New adapter `internal/mcpserver`. It may import `internal/app`, `internal/domain`, and the go-sdk, and never `internal/store` or `internal/ui`. Ports (`AssistantConnection`, `DesktopConfirmer`, `ConfirmationRouter`, `ChangeEvent`, and the repository contracts) live in `internal/app`, so `fyneui` and `mcpserver` never import each other. Wiring happens only in `cmd/fallout-tracker`. The Principle I diagram is amended to show the new adapter (R18). |
| II. Domain rules before UI | Pass | No new combat rules. Draft-roster expansion and the difficulty preview live in `internal/app` and reuse `domain.EvaluateEncounterDifficulty`, `ValidateCombatant`, `prepareEncounterCombatants`, and the `Encounter.ApplyDamage` / `Heal` / `Resources` methods. The adapter validates only the wire format. |
| III. Migrations | Pass | Migration `00044` is reversible. The plan includes the `schema.sql` and `query.sql` updates, dbgen regeneration, `docs/db` regeneration, and a `docs/db-normalization.md` note on why `source` is a CHECK-constrained column. |
| IV. Tests | Pass | Domain: settings and port validation. App: gate, revisions, change events, source tagging, draft roster, activity. Store: migration up/down, repositories, retention. Adapter: tools, parity, admission, confirmation, skills, bridge. UI: dialogs, refresh, stale-edit flows. |
| V. Generated code | Pass | `dbgen` comes only from `make sqlc-generate`. |
| VI. Local desktop first | **Pass with amendment** | This spec explicitly introduces a loopback-only, off-by-default, token-protected endpoint. No internet access, remote hosting, or accounts are involved. The constitution text is amended to name the exception (R18). |
| VII. Audit side effects | Pass | Both encounter-log and assistant-activity writes run through `runNonCriticalSideEffect`. Their failure never fails the action. |
| Tooling rule (`tools/<tool>` modules) | Pass | No new tool modules. The go-sdk is an application dependency, not a tool. |
| Quality gates | Pass | Listed in `quickstart.md` §1. |

## Project Structure

### Documentation (this feature)

```text
specs/004-mcp-server/
├── spec.md
├── plan.md                    # this file
├── research.md                # R1–R20 decisions
├── data-model.md              # settings, activity, log source, revisions, events
├── quickstart.md              # gates + manual validation
├── checklists/
│   └── requirements.md
├── contracts/
│   ├── mcp-tools.md           # 30 tools, DTOs, errors, confirmation, resources
│   ├── mcp-connection.md      # endpoint, admission, capabilities, skills, bridge, client setup
│   └── desktop-ui.md          # ASSISTANT dialog, confirm dialog, refresh, stale edits, DATA marker
└── tasks.md                   # created by /speckit-tasks
```

### Source Code (repository root)

```text
.specify/memory/constitution.md          # v1.2.0 amendment (Principle I diagram, VI exception, structure rules)

cmd/fallout-tracker/
└── main.go                              # subcommand dispatch (GUI | mcp-bridge); `version` var (ldflags, BuildInfo fallback); wiring store→service→router→mcpserver→fyneui

internal/domain/
├── assistant.go                         # AssistantConnectionSettings (+ port validation, redacted secret), AssistantActivityEntry, outcomes, entity kinds
├── encounter.go                         # EncounterLog.Source (OperationSource)
├── errors.go                            # ErrTrackerBusy, ErrStaleEdit, ErrEntityDeleted
└── assistant_test.go

internal/app/
├── gate.go                              # weighted-semaphore operation gate (read/write, ctx-aware)
├── source.go                            # WithSource / SourceFromContext
├── changes.go                           # ChangeEvent, SubscribeChanges, publish after release
├── revisions.go                         # EntityKey, Revision(), bump/tombstone, ExpectedRevision checks
├── encounter_draft.go                   # EncounterDraft expansion, PreviewEncounterDifficulty, Create/UpdateFromDraft
├── campaign_roster_patch.go             # ExecuteCampaignRosterPatch (read-patch-write under one gate hold)
├── assistant_service.go                 # settings get/save/regenerate, activity append/list/clear
├── assistant_ports.go                   # AssistantConnection, ConnectionStatus, DesktopConfirmer, ConfirmationRequest/Decision, ConfirmationRouter
├── commands.go                          # ExpectedRevision on Update{Campaign,Encounter}Command
├── operation_log.go                     # source-aware appendOperationLog
├── repository_contracts.go              # AssistantSettingsRepository, AssistantActivityRepository; AppendEncounterLog(+source)
├── service.go                           # gate + revisions + subscribers fields; public methods wrapped
└── *_test.go                            # gate (incl. -race), revisions, events, draft parity, activity, source tagging

internal/store/sqlite/
├── db.go                                # _pragma=busy_timeout(5000)
├── migrations/00044_add_assistant_connection.sql
├── sqlc/schema.sql                      # regenerated
├── sqlc/query.sql                       # settings get/upsert; activity insert/prune/list/clear; log insert with source; list logs with source
├── dbgen/                               # regenerated
├── assistant_repository.go              # settings + activity repositories
├── encounter_logs_repository.go         # source column
├── mappers.go                           # log source mapping
└── *_test.go                            # migration up/down, retention, defaults/backfill

internal/mcpserver/                      # NEW adapter
├── server.go                            # mcp.Server construction, options (instructions, cacheable, capabilities)
├── connection.go                        # Connection: Start/Stop/Status/Apply; net.Listen 127.0.0.1; http.Server
├── admission.go                         # refuseBrowser, requireLoopbackHost, bearer verifier (atomic secret), refusal coalescing
├── tools_read.go                        # read-only tools
├── tools_campaign.go                    # campaign tools (patch-style)
├── tools_encounter.go                   # encounter tools incl. draft/update/restart/delete
├── tools_combat.go                      # advance_turn, apply_damage, heal, change_party_ap, change_gm_threat
├── dto.go                               # DTOs + domain↔DTO mapping
├── errors.go                            # domain/app error → tool error codes, by error kind (errors.Is on domain.ErrInvalidInput / ErrNotFound; research R22)
├── confirm.go                           # MRTR elicitation, HMAC requestState, desktop fallback via app.DesktopConfirmer
├── activity.go                          # per-tool activity recording
├── skills.go                            # SEP-2640 skills/list, skills/get, digests; resources registration
├── skills/{start-campaign,prepare-encounter,run-combat-round}/SKILL.md
├── reference/rules.md                   # fallout://reference/rules
├── bridge.go                            # stdio ⇄ HTTP relay for `mcp-bridge`
└── *_test.go                            # httptest + go-sdk client; parity; admission; confirm; skills; bridge

internal/ui/fyneui/
├── app.go                               # Run(ctx, svc, onShutdown, Options); change subscription; StartIfEnabled
├── main_screen.go / layout.go           # ASSISTANT header button + status
├── assistant_connection_dialog.go       # NEW
├── assistant_confirm_dialog.go          # NEW: DesktopConfirmer implementation (FIFO, 2-min deadline)
├── assistant_refresh.go                 # NEW: coalesced fyne.Do refresh on assistant events
├── encounter_list_dialog.go             # reload on events, keep selection
├── encounter_editor_dialog.go           # capture revision; stale/deleted handling
├── campaign_dialogs.go                  # capture revision; stale handling
├── data_log.go                          # [AI] marker
├── shutdown.go                          # Stop connection before Quit
└── *_test.go

docs/
├── db/                                  # regenerated
├── db-normalization.md                  # source column rationale
└── assistant.md                         # NEW: GM setup guide (Claude Code, Claude Desktop, troubleshooting)
README.md                                # short "Assistant (MCP)" section linking docs/assistant.md
CLAUDE.md                                # Project Boundaries: add internal/mcpserver (outside SPECKIT markers)
.goreleaser.yaml / .goreleaser.darwin.yaml / Makefile   # -ldflags "-X main.version=..." (research R21)
```

**Structure Decision**: add a single new inbound adapter package, `internal/mcpserver`, at the same level as `internal/ui/fyneui`. Every cross-adapter interaction goes through ports declared in `internal/app`, so the existing `domain ← app ← {store, ui}` direction is kept. MCP becomes one more `app` client.

## Data Model / Migration Plan

See `data-model.md` for the field definitions.

1. **`00044` Up**:
   - Create `assistant_connection_settings`: a single row with `enabled`, `port`, `access_secret`, and timestamps. The row is created by the application on first read, not seeded by the migration, because the migration cannot produce a random secret.
   - Create `assistant_activity` with its `occurred_at DESC` index.
   - Add `encounter_logs.source TEXT NOT NULL DEFAULT 'desktop' CHECK (source IN ('desktop','assistant'))`. Existing rows backfill to `desktop`.
2. **`00044` Down**:
   - Drop the two tables.
   - Rebuild `encounter_logs` without `source`, keeping its rows, its foreign key, its checks, and the `idx_encounter_logs_encounter_created` index.
3. **Queries**:
   - `GetAssistantSettings`, `UpsertAssistantSettings`.
   - `InsertAssistantActivity`, `PruneAssistantActivity` (keep the newest 1,000), `ListAssistantActivity(limit)`, `ClearAssistantActivity`.
   - `InsertEncounterLog` gains `source`. `ListEncounterLogsByEncounterID` selects `source`.
4. **Regenerate** `schema.sql`, `dbgen`, and `docs/db`. Add a `docs/db-normalization.md` note.

## Implementation Phases

**P0. Governance and spike.**

- Amend the constitution to v1.2.0 (R18).
- Add the go-sdk v1.8.0 dependency.
- Spike the stateless handler with a bridge relay: a legacy-era go-sdk client talks over stdio to the relay, which talks to an httptest endpoint. Check `initialize` followed by `tools/call`. This removes the main protocol-era risk (R8) before the full build-out.

**P1. App foundations** (blocking for everything else).

- Add the operation gate (R9) at every exported `Service` method. Collapse wrapper pairs so each takes the gate once.
- Add source context (R12), change events (R10), and revisions with `ExpectedRevision` (R11).
- Add the new domain errors.
- Add the `busy_timeout` pragma.
- Tests: concurrency with `-race`, no deadlock on wrapper pairs, revision bump rules, and events published after release.

**P2. Persistence.**

- Migration `00044`, the queries, and the repositories for settings, activity (with retention) and the log source.
- Regenerate.
- Tests: up/down migration, backfill, retention at 1,001 rows, the secret never logged.

**P3. App use cases.**

- `assistant_service.go`: settings with lazy generation, port validation, secret regeneration, activity append/list/clear.
- `encounter_draft.go`: preview plus create/update from a draft (R14).
- `campaign_roster_patch.go` (R15).
- Tests: parity between the draft expansion and the editor collector semantics, and patch behavior (rename gives a new ID, removal, availability).

**P4. MCP adapter, MVP = US1 + US2 + US3.**

- `server.go` and `connection.go` (listener lifecycle, port-in-use status).
- `admission.go` (R3), `dto.go`, `errors.go`, read tools, campaign tools, encounter tools.
- Error codes come from the error kind, never from message wording (FR-025, research R22):
  - `internal/domain` defines `ErrInvalidInput` and `ErrNotFound`, plus `InvalidInput(err)` and `NotFound(err)`, which keep the message;
  - `internal/app` marks its checks, its validation helpers and the results of domain rules, and the store marks its user-caused guards;
  - the adapter maps by `errors.Is`, and anything unmarked is `internal`.
- `confirm.go` (R5) with the router, and `activity.go`.
- Tests:
  - each tool over httptest;
  - parity against direct service calls on two fresh databases;
  - admission matrix (no token, bad token, Origin, `Sec-Fetch-*`, bad Host);
  - MRTR accept, decline, forged, reused, and mismatched-arguments `requestState`;
  - desktop fallback with a fake confirmer: accept, decline, timeout, cancel;
  - activity outcomes.

**P5. Live combat (US4).**

- The combat tools.
- Tests: damage by type and location with DR and immunity, torso-only creatures, the AP cap, overspend rejection, and "no active encounter".

**P6. Desktop integration (US6).**

- `fyneui.Run` options, the ASSISTANT button and dialog, the confirmation dialog, coalesced refresh, encounter-list reload, editor revisions with stale and deleted handling, the `[AI]` DATA marker, and shutdown.
- Wiring in `cmd/fallout-tracker/main.go`.
- Tests: dialog state, refresh coalescing (fake clock), stale-edit flows with a fake service response, FIFO confirmer behavior.

**P7. Skills and reference content (US5).**

- `skills.go`, the three `SKILL.md` files, `reference/rules.md`, and the server instructions.
- Tests: manifest digests match the served bytes, `skills/get` for an unknown skill returns -32602, frontmatter is valid, and resources are listed for clients without the extension.

**P8. Bridge and docs.**

- `bridge.go` and the `mcp-bridge` subcommand. Error mapping: -32000 when unreachable, -32001 for a 401.
- `docs/assistant.md` and the README section.
- Tests: the relay handles notifications, request/response matching, header derivation, and exit codes.

**P9. Verification.**

- The full gates and the manual checks in `quickstart.md` §§2–5, in both Claude Code and Claude Desktop (SC-011).
- Record the results in the final implementation report.

## Design Artifacts

- `research.md`: decisions R1–R22 (SDK, stateless transport, admission security, secret storage, confirmation, skills, fallback resources, Desktop bridge, the operation gate, live refresh, stale edits, source tagging, activity log, draft rosters, campaign patch tools, lifecycle, wiring, constitution amendment, exclusions, preserved behaviors).
- `data-model.md`: the new tables and column, domain types, in-memory revisions, events, and confirmation requests.
- `contracts/mcp-tools.md`: the tool catalog, DTOs, error codes, confirmation flow, and resources.
- `contracts/mcp-connection.md`: the endpoint, admission order, capabilities, skills wire format, bridge CLI, and client setup snippets.
- `contracts/desktop-ui.md`: the affected Fyne screens and dialogs, refresh rules, stale-edit UX, and shutdown.
- `quickstart.md`: generation, automated gates, and manual validation for both clients.

**Bugfix**: 2026-11-27 — BUG-001 Updated from bugfix patch: error kinds in P4 and in the `errors.go` entry (FR-025, research R22).

## Post-Design Constitution Check

| Principle | Status | Evidence |
|---|---|---|
| I. Layer boundaries | Pass (with the v1.2.0 amendment as task P0) | The structure tree has no `mcpserver → store/ui` or `ui → mcpserver` imports. Ports are in `internal/app/assistant_ports.go`. Wiring is only in `cmd`. |
| II. Domain rules | Pass | Contracts mark adapter validation as wire-format only. Rules stay in domain/app (data-model §Validation rules reused). |
| III. Migrations | Pass | Migration `00044` Up/Down, queries, regeneration, and docs are named. |
| IV. Tests | Pass | Each phase lists its tests. Quickstart §1 adds `-race` for the concurrency-sensitive packages. |
| V. Generated code | Pass | Regeneration steps are in P2 and in quickstart §1, with a `git diff --exit-code` check. |
| VI. Local desktop | Pass (with the v1.2.0 amendment) | Loopback bind, off by default, secret, Origin and Host checks. The bridge talks only to `127.0.0.1`. |
| VII. Audit side effects | Pass | Activity and encounter-log writes are non-critical (`runNonCriticalSideEffect`, new ID `audit.append_assistant_activity`). |
| Quality gates | Pass | `go test ./...`, `go vet ./...`, `make lint`, `make build`, `make sqlc-generate`, `make db-doc-generate`. |

## Complexity Tracking

| Addition | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Loopback HTTP listener inside the desktop process | Chosen in Clarifications (embedded runtime). MCP 2026-07-28 features (skills, MRTR) need stateless HTTP. | stdio (a process launched by the client) cannot see the open app's live state and was rejected by the user. A Unix socket is not supported by either client. |
| Hand-rolled Skills extension (`skills/list`, `skills/get`) | FR-018 requires server-shipped skills, and no tagged go-sdk release includes the extension. | Pinning a go-sdk `main` pseudo-version means depending on an unreleased API with no stability guarantee. The code is isolated behind an interface so it can switch to the SDK package once that ships. |
| Built-in stdio bridge subcommand | Claude Desktop Chat cannot reach `127.0.0.1` (connectors run from Anthropic's cloud, and local servers are stdio only). FR-008 requires Desktop support. | `mcp-remote` needs Node.js on the GM's machine. It is still documented as an alternative. |
| Service-wide operation gate + in-memory revisions | A second concurrent caller (the assistant) makes the existing read-modify-write paths lose updates, and FR-022 requires conflict warnings. | DB versioning needs schema and query changes in every write path. `SetMaxOpenConns(1)` does not make read-modify-write atomic. |
