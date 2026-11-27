# Quickstart: Validate The Assistant Connection

This guide shows how to check the feature end to end. Behavior contracts are in [`contracts/`](contracts/), and the new data is described in [`data-model.md`](data-model.md).

## Prerequisites

- Go 1.27.4 and the repository tools: `make lint-install`, `make sqlc-install`, `make tbls-install`.
- Claude Code v2.1.332 or later, for the skills checks.
- Claude Desktop, for the bridge checks.
- A scratch database, so real campaign data stays untouched:

  ```bash
  export FALLOUT_TRACKER_DB_PATH="$(mktemp -d)/tracker.db"
  ```

## 1. Generation and automated gates

```bash
make sqlc-generate                 # after migration 00044 and the query.sql changes
make db-doc-generate               # regenerates docs/db
git diff --exit-code internal/store/sqlite/dbgen docs/db   # must be clean after regeneration
go test ./internal/domain/... ./internal/app/... ./internal/store/sqlite/... ./internal/mcpserver/... ./internal/ui/fyneui/...
go test -race ./internal/app/... ./internal/mcpserver/...  # gate, change events, confirmations
go test ./...
go vet ./...
make lint
make build
```

Expected results:

- All commands pass.
- The tests in `internal/mcpserver` cover:
  - each tool against a temp-dir SQLite service, using an `httptest` server and a go-sdk client;
  - parity with direct `app.Service` calls (SC-002);
  - admission: Origin, Host and secret (SC-008);
  - MRTR confirmation, including a forged or reused `requestState`;
  - the desktop confirmation fallback, with a fake confirmer;
  - activity log outcomes (SC-010);
  - the skills manifest digests;
  - a bridge relay test (legacy client → stdio bridge → stateless endpoint).

## 2. Manual: enable the connection (US6)

1. Run `./bin/fallout-tracker`. The header shows `ASSISTANT · OFF`.
2. Run `curl -s -o /dev/null -w '%{http_code}' -X POST http://127.0.0.1:47823/mcp`. **Expected:** `000`, connection refused (SC-008: off by default).
3. Open ASSISTANT and turn the connection on. **Expected:** `Running at http://127.0.0.1:47823/mcp`.
4. Run the same `curl` without a token. **Expected:** `401`.
5. Run it again with `-H 'Origin: http://evil.example'`. **Expected:** `403`.
6. Run it again with the token and `-H 'Origin: http://127.0.0.1:47823'`. **Expected:** not `403`. The endpoint's own loopback origin is allowed.
7. Run `lsof -nP -iTCP:47823 -sTCP:LISTEN` (on Windows, `netstat -ano | findstr 47823`). **Expected:** the listener is on `127.0.0.1` only (SC-008).
8. Open the activity list. **Expected:** two `refused` entries, `bad_secret` and `browser_origin`.
9. Restart the app. **Expected:** the connection comes back on (FR-003).

## 3. Manual: Claude Code (US1–US5)

1. Copy the Claude Code snippet from the ASSISTANT dialog and run it. Then run `claude mcp list`. **Expected:** `fallout-tracker` is connected.
2. In `claude`, ask: "Create a campaign 'Wasteland' starting today with players Ann (Nora, level 3) and Bob (Dogmeat, level 2)". **Expected:** the CAMP tab shows the campaign within 1 second (SC-004).
3. Ask: "/fallout-tracker:prepare-encounter a hard fight for my party". **Expected:**
   - the skill loads;
   - the assistant calls `preview_encounter_difficulty` until it gets `Hard`, then creates the encounter;
   - the encounter list matches the label.
4. Ask: "Activate it, advance the turn, deal 7 physical damage to the torso of the first raider, add 2 Threat". **Expected:**
   - the STAT view updates live;
   - DATA shows these entries with `[AI]`;
   - the reported `applied` damage matches the DR.
5. Ask: "Delete that encounter". **Expected:**
   - a terminal elicitation form appears;
   - answering **Decline** leaves the encounter in place, and the activity list shows `declined`;
   - repeating and choosing **Accept** deletes it.
6. In a new `claude` session, ask: "Undo the damage you dealt last". **Expected:** the assistant reads `list_assistant_activity` and heals the same amount.

## 4. Manual: Claude Desktop through the bridge (SC-006, SC-011)

1. Copy the Claude Desktop snippet into `claude_desktop_config.json` and restart Claude Desktop. **Expected:** the tracker tools are listed.
2. Ask it to restart the active encounter. **Expected:**
   - the **tracker window** shows "Assistant wants to…";
   - choosing Cancel returns `declined`;
   - leaving the dialog alone for 50 seconds returns `timed_out` (the dialog shows a countdown);
   - Claude Desktop does not cut the request before then. If it does, record after how many seconds in the validation log: the 50-second window must stay below that limit (research R5).
3. Quit the tracker and ask again. **Expected:** Claude reports that the tracker is not running (bridge `-32000`).

## 4a. Measured success criteria (trial protocol)

Record every trial in the "Validation log" section at the end of this file.

| Criterion | Trials | Procedure | Pass if |
|---|---|---|---|
| SC-004 | 20 | With STAT open, have Claude Code run one modifying tool (alternate `apply_damage`, `heal`, `advance_turn`, `update_campaign_details`). Measure from the tool result appearing in the terminal to the desktop view showing the change, using a stopwatch or screen recording. | At least 19 of 20 trials ≤ 1 s, and no unsaved dialog input lost in any trial |
| SC-009 | 5 | Fresh Claude Code session each time: "/fallout-tracker:prepare-encounter a hard fight for my party" | At least 4 of 5 trials end with a saved encounter labelled `Hard` |
| SC-001 | 1 | Time quickstart §3 steps 2–3 from an empty database | Under 5 minutes |
| SC-006 | 1 per client | Time the setup from "connection off" to a working `list_campaigns`: Claude Code, then Claude Desktop with the bridge | Under 5 minutes (Code) and under 10 minutes (Desktop) |

## 5. Manual: concurrency and stale edits (US6 acceptance scenarios 4–6)

1. In the tracker, open the encounter editor for encounter E and change a name, but do not save.
2. Ask the assistant to rename E, then save in the tracker. **Expected:** the Reload/Overwrite prompt appears, and neither choice loses data silently.
3. Open the editor for E again, ask the assistant to delete E, then save. **Expected:** a close-only notice, and E is not recreated.
4. Ask the assistant to "deal 1 damage to every NPC, one call each" while you click Next Turn repeatedly. **Expected:** no `database is locked` errors, the final HP equals the starting HP minus the applied damage, and the round and turn match the number of clicks.

## 6. Clean up

```bash
claude mcp remove fallout-tracker
rm -rf "$(dirname "$FALLOUT_TRACKER_DB_PATH")"
```

Remove the `fallout-tracker` entry from `claude_desktop_config.json`.

## Validation log

Filled in during T088. Record each step and trial from §§2–4a as pass or fail, with the date, the client versions, and any timings.

### 2026-11-27: automated gates and headless smoke run (implementation agent)

Environment: macOS (darwin/arm64), Go 1.27.4, go-sdk v1.8.0, Claude Code 2.1.326. The endpoint was served headless (a temporary test running `mcpserver.Connection` on `127.0.0.1:47898` over a seeded temp database), not through the GUI.

| Check | Result |
|---|---|
| §1 `make sqlc-generate`, `make db-doc-generate` | pass; regenerated `dbgen`, `sqlc/` and `docs/db` are byte-identical to the working tree (no drift) |
| §1 `go test -count=1 ./...`, `go vet ./...`, `make lint` (0 issues), `make build`, `gofmt -l` | pass (exit 0) |
| §1 `go test -race ./internal/app/... ./internal/mcpserver/...` | pass (app 300 s, mcpserver 396 s, no data races) |
| §2 no token → 401; foreign `Origin` → 403 (with or without a valid token); rebinding `Host` → 403 | pass |
| §2 listener bound to `127.0.0.1` only (`lsof`) | pass |
| §2 refused requests appear in the activity log, one entry per reason (`bad_host`, `browser_origin`, `bad_secret`) | pass |
| §3 step 1: Claude Code `claude mcp add --transport http … --header` then `claude mcp list` | pass: `✔ Connected` (protocol 2026-07-28, no `Origin`/`Sec-Fetch-*` headers, research R3) |
| §4 bridge binary `fallout-tracker mcp-bridge` with a 2025-11-25 client: initialize, tools/list (30 tools), tools/call | pass |
| §4 bridge with the tracker down → `-32000` "not running"; missing secret → exit 2 | pass |

**Not run (needs a person at the GUI or Claude Desktop):** §2 steps 1, 3 and 9 through the desktop app (toggle, restart persistence), §3 steps 2–6 in an interactive Claude Code session (elicitation form, `/fallout-tracker:prepare-encounter` skill, live STAT refresh), §4 with Claude Desktop itself (desktop confirmation dialog, its request timeout versus the 50 s window), §4a timed trials (SC-001, SC-004, SC-006, SC-009) and §5 stale-edit and concurrency checks in the GUI. These behaviors are covered by automated tests (`internal/ui/fyneui/*_test.go`, `internal/mcpserver/*_test.go`), but the manual pass in both clients is still open.

### 2026-11-28: manual pass in Claude Code and Claude Desktop (GM)

Environment: macOS (darwin/arm64), the tracker built from branch `004-mcp-server`, Claude Code 2.1.332 (native install, latest channel), Claude Desktop 6.28899.6. Another local app held port 47823, so the connection ran on 47824. The run started from an empty database.

The GM reported every check below as passing. Per-trial timings and counts were not itemized.

| Check | Result |
|---|---|
| §2: off by default, enabling, 401/403 admission, loopback-only listener, refusals in the activity list, restart keeps the connection on | pass |
| Failed start on a busy port: turning the connection off clears the failure; APPLY with a free port recovers (FR-006) | pass |
| §3 in Claude Code: campaign, `/fallout-tracker:prepare-encounter` skill (the 2.1.326 stable build had no MCP skills; skills arrived in 2.1.332), live STAT and DATA `[AI]`, terminal elicitation Decline/Accept, undo from activity | pass |
| §4 in Claude Desktop through the bridge: tracker-window confirmation (Cancel → `declined`, 50 s → `timed_out`), tracker down → "not running", §3 scenarios through resources | pass |
| §4a: SC-001, SC-004, SC-006 (Code and Desktop), SC-009 | pass |
| §5: Reload/Overwrite on a stale edit, close-only notice for a deleted encounter, no `database is locked` under concurrent turns | pass |
| Stopping the connection, or pressing Stop in Claude Desktop, while a desktop confirmation is open declines it | pass |
| Removing a player in the campaign editor also removes their character from the campaign's encounters (baseline 001 US1/AC3) | pass |
| Duplicate player names (`Ann`/`ann`) rejected from the assistant and from the editor | pass |
