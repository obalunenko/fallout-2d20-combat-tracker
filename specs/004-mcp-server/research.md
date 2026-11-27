# Research: Assistant Control Of Campaigns And Encounters (MCP Server)

Date: 2026-11-27. Sources are primary: the modelcontextprotocol.io spec and extension pages, `github.com/modelcontextprotocol/go-sdk` at tag v1.8.0, code.claude.com/docs, and claude.com/docs/connectors. Codebase facts cite `file:line` as of commit `6f941bc`.

## R1. MCP library

- **Decision**: `github.com/modelcontextprotocol/go-sdk` **v1.8.0**, the latest stable release (tagged 2026-09-14, `go 1.25.0`). It is added to the application `go.mod`.
- **Rationale**:
  - It is the official SDK.
  - Its `latestProtocolVersion` is `2026-07-28`, and it still accepts 2025-11-25 and 2025-06-18.
  - It provides the pieces this feature needs: typed `mcp.AddTool[In, Out]` with inferred input and output schemas, `ToolAnnotations`, MRTR elicitation (`CallToolResult.InputRequests` / `RequestState`), resources, `Capabilities.AddExtension`, `AddReceivingCustomMethod`, and the `auth.RequireBearerToken` middleware.
- **Alternatives considered**:
  - **`github.com/mark3labs/mcp-go` v1.6.0**: roughly the same core features, but it has no auth helpers and no Skills support either. Rejected in favor of the official SDK.
  - **go-sdk v1.9.0-pre.1**: a pre-release. Its only relevant fix (#1266) affects stateful sessions, and this design does not use them.
  - **go-sdk `main`**: has an unreleased `extensions/skills` package. Rejected for now (see R6).
- **Dependency note**: the SDK pulls in `github.com/google/jsonschema-go` and `github.com/yosida95/uritemplate` transitively. `make tidy` and `go mod verify` cover them. Tool dependencies are unaffected: they stay in `tools/*`.

## R2. Transport and protocol era

- **Decision**:
  - Streamable HTTP at `http://127.0.0.1:<port>/mcp`, created with `mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true})`.
  - One `*mcp.Server` instance is shared by all requests.
  - The default port is `47823`. The GM can change it.
- **Rationale**:
  - On HTTP, the SDK serves the 2026-07-28 revision **only** when `Stateless` is true. A stateful handler pushes clients down to 2025-11-25.
  - Claude Code's MCP skills need 2026-07-28. Its v2 runtime asks servers for the newer revision.
  - Spec 2026-07-28 removes sessions, `initialize`, and server-initiated requests. A stateless handler matches that model, and it still answers legacy requests so the stdio bridge can relay them (R8).
  - `JSONResponse` keeps the bridge relay simple: one JSON body per POST.
  - `PropagateRequestCancellation` turns a client-side cancel into a declined confirmation (R5).
- **Consequences**:
  - GET and DELETE return 405.
  - Legacy-era clients can call tools but cannot be elicited, so confirmation falls back to the desktop dialog (R5).
  - List and read results must carry `ttlMs` and `cacheScope`. Set them with `ServerOptions.SetCacheable`, so tool, resource, and skill lists are cacheable.
- **Alternatives considered**:
  - **stdio**: a process launched by the client. The user rejected it in Clarifications: the connection is embedded in the desktop app.
  - **Stateful HTTP**: loses 2026-07-28, and with it Claude Code skills.
  - **Unix domain socket**: Claude Code and Claude Desktop cannot connect to one.

## R3. Local endpoint security (FR-005, SC-008)

- **Decision**: the listener binds to `127.0.0.1` only (`net.Listen("tcp", "127.0.0.1:<port>")`). Requests pass this middleware chain, outermost first:
  1. **`refuseBrowser`**: respond 403 when `Origin` is present and is not exactly `http://127.0.0.1:<port>` or `http://localhost:<port>`, or when `Sec-Fetch-Site` is present with a value other than `none` or `same-origin`. Record an activity refusal `browser_origin`.
     - Non-browser MCP clients normally send neither header. An `Origin` equal to the endpoint's own loopback origin is allowed, so a client that sets it anyway is not locked out. No web page can carry that origin, because nothing else is served on that port.
     - T004 records the headers a real Claude Code connection sends, and the result is noted here before T032 is implemented.
     - **T004 result (2026-11-27, Claude Code 2.1.326)**: a `claude mcp list` health check sent `POST /mcp`. The headers were `Host=127.0.0.1:<port>`, `Accept=application/json, text/event-stream`, `Authorization` (bearer), `Content-Type=application/json`, `Mcp-Method=server/discover`, `Mcp-Protocol-Version=2026-07-28` and `User-Agent=claude-code/2.1.326 (sdk-cli)`. There was **no `Origin` and no `Sec-Fetch-*`**. The policy above holds unchanged.
  2. **`requireLoopbackHost`**: respond 403 unless `Host` is `127.0.0.1:<port>` or `localhost:<port>`. This blocks DNS rebinding. Record the refusal `bad_host`. The SDK runs the same check by default, which stays on as defence in depth.
  3. **`auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})`**: the verifier compares the token with the current secret using `subtle.ConstantTimeCompare`. A mismatch returns `auth.ErrInvalidToken` and a 401. Record the refusal `bad_secret`.
     - **As built**: `internal/mcpserver/admission.go` checks the bearer token itself, with the same `subtle.ConstantTimeCompare`, a 401 and `WWW-Authenticate: Bearer`. The SDK middleware answers a request without a token before it calls the verifier, so that refusal could not be recorded as `bad_secret`.
  4. The SDK handler.
- **Rationale**:
  - Spec 2026-07-28 Streamable HTTP says: "MUST validate the `Origin` header… present and invalid → 403", "SHOULD bind only to localhost", and "SHOULD implement proper authentication". The security best-practices page recommends an authorization token for local HTTP servers.
  - The SDK's built-in protection checks `Host` only, not `Origin`.
  - `http.CrossOriginProtection` allows same-origin requests, so it cannot stop DNS rebinding.
  - The access secret itself is 32 bytes from `crypto/rand`, base64url-encoded. It is stored in the tracker database (R13). Rotation swaps it in an `atomic.Pointer` held by the verifier, so the next request already sees the new value.
- **Alternatives considered**:
  - **OAuth (the MCP authorization spec)**: overkill for a single local user, and the clients would need an authorization server.
  - **No token**: the user rejected it in Clarifications.

## R4. Access secret storage

- **Decision**: store the secret in the single row of `assistant_connection_settings` in tracker.db, together with `enabled` and `port` (see data-model.md).
- **Rationale**:
  - The app has no Fyne app ID (`fyneapp.New()`, `internal/ui/fyneui/app.go:16`), so `Preferences()` would be in-memory only.
  - The tracker database is already the app's single durable store, honors `FALLOUT_TRACKER_DB_PATH`, and isolates tests.
  - The secret protects only the GM's own local data, which is in that same file. Storing it next to the data adds no exposure.
- **Alternatives considered**:
  - **The OS keychain**: platform-specific code for little gain.
  - **A separate config file**: a second store to keep in sync.
  - **Setting a Fyne app ID**: changes where Fyne stores app data, with no other benefit.

## R5. Confirmation of destructive operations (FR-015)

- **Decision**: the adapter confirms before it calls the application service, and never holds the operation lock while waiting.
  - **Elicitation path**: used when `req.ProtocolVersion() >= "2026-07-28"` and `req.ClientCapabilities().Elicitation` (form) is declared.
    1. The first call returns `CallToolResult{InputRequests: {"confirm": ElicitParams{Mode: "form", Message, RequestedSchema: {confirm: boolean}}}, RequestState: <token>}`.
    2. The client retries with `InputResponses["confirm"]`.
    3. The adapter proceeds only on `Action == "accept" && Content["confirm"] == true`.
  - **Desktop path**: used in every other case.
    1. The adapter calls `app.DesktopConfirmer.Confirm(ctx, ConfirmationRequest)` and blocks the HTTP request for up to **50 seconds**.
       - Many MCP clients cut a request after 60 seconds: that is the TypeScript SDK default request timeout, and Claude Desktop's value is unverified.
       - In stateless JSON-response mode the server cannot send progress notifications to extend the deadline, so the window must stay below 60 seconds.
       - The bridge sets no client-side timeout below 150 seconds.
    2. The Fyne implementation queues requests first in, first out, and shows one dialog at a time, with a countdown, through `fyne.Do` / `dialog.NewCustomConfirm`.
    3. A client cancel cancels the context, which closes the dialog and counts as declined. An unanswered dialog returns `timed_out`, and the operation does not run.
  - **The `RequestState` token**:
    - It is an HMAC-SHA256 keyed by a per-process random key, minted with `crypto/rand` at startup.
    - It covers the tool name, a digest of the canonical arguments, an expiry 2 minutes out, and a nonce.
    - It is single-use: the nonce is recorded and rejected if reused. A used nonce is kept until the token's expiry plus 1 minute, so a token replayed shortly after it expired is still recognized as a reuse (`declined`/`rejected`), not a late answer (`timed_out`). After that grace period, the token's own expiry rejects any replay as `timed_out`, and the operation never runs.
    - Verification separates the failure cases. An authentic token (HMAC valid, arguments digest matches, nonce unused) past its expiry gives `timed_out`. A bad HMAC, a reused nonce, or an arguments mismatch gives `declined`, recorded as `rejected`. The operation never runs in either case. `InputResponses` sent without any `requestState` start a fresh confirmation.
  - **The dialog text** is built from stored state, such as the encounter name looked up by ID. It never uses free text from the assistant, so a request cannot mislabel or pre-answer its confirmation.
- **Rationale**:
  - Under 2026-07-28, MRTR is the only way to elicit (SEP-2322). Servers "MUST NOT send `inputRequests` the client has not declared".
  - `requestState` is attacker-controlled, so the MRTR pattern page asks for HMAC protection, a short TTL, binding to the request, and single use.
  - The desktop dialog is the fallback chosen in Clarifications. It works for any client, including Claude Desktop Chat through the bridge, where elicitation support is undocumented.
  - Holding the operation lock during a confirmation wait (up to 50 s on the desktop, or the client's think time with elicitation) would block the UI. The Fyne dialog runs on the main goroutine, so holding the lock could even deadlock.
  - The existence check repeats inside the service call. If the entity was deleted meanwhile, the call returns `not_found`.
- **Alternatives considered**:
  - **A two-phase `confirm_token` argument the assistant passes back**: the user rejected it because the assistant could confirm on its own.
  - **`_meta["anthropic/requiresUserInteraction"]`**: Claude Code only, and it duplicates the prompt.

## R6. Skills over MCP (FR-018)

- **Decision**: implement the Skills extension (SEP-2640, `io.modelcontextprotocol/skills`) by hand in `internal/mcpserver/skills.go`:
  - Advertise it with `ServerOptions.Capabilities.AddExtension("io.modelcontextprotocol/skills", map[string]any{"directoryRead": false})`, alongside `resources`.
  - Register `skills/list` and `skills/get` with `mcp.AddReceivingCustomMethod`. Results embed `ResultBase` and carry `resultType: "complete"`, `ttlMs`, and `cacheScope`.
  - Each skill is an embedded `SKILL.md` (`embed.FS`) served at `skill://<name>/SKILL.md` through the normal `resources/read`. The manifest digest (`sha256:…`) and size are computed from the embedded bytes at startup.
  - **Skills shipped**: `start-campaign`, `prepare-encounter`, and `run-combat-round`. Frontmatter: `name` (equal to the directory name), `description`, and `metadata.version` set to the tracker version.
  - Keep the code behind a small interface so it can switch to the SDK's `extensions/skills` package once that ships in a tagged release.
- **Rationale**:
  - No released Go SDK includes the extension. It was merged to `main` on 2026-11-27, which requires Go 1.26, and its API is unreleased.
  - The wire surface is small: two methods plus ordinary resources.
  - Claude Code (v2.1.332+) loads MCP skills on 2026-07-28 connections.
  - The skills ship inside the binary, so upgrading the tracker upgrades the skills with no install step (US5 acceptance scenario 3).
- **Alternatives considered**:
  - **Pinning a go-sdk `main` pseudo-version**: unreleased API churn.
  - **mark3labs/mcp-go**: no Skills support.
  - **Plugin-installed skills instead of MCP skills**: needs a separate install, which FR-018 forbids.

## R7. Reference content for clients without skills (FR-019, FR-020)

- **Decision**:
  - Register the same `skill://…/SKILL.md` URIs as ordinary resources in `resources/list`. The Skills spec says that to a client without the extension they "are ordinary resources".
  - Add `ServerOptions.Instructions`, a short text of under 1,500 characters. It lists the three guides by URI, states the key rules (the 6-point Party AP cap, the difficulty bands, the destructive operations that need GM confirmation), and points out that `update_encounter` makes the edited encounter active.
  - Every tool description explains the Fallout 2d20 terms it uses (FR-020).
- **Rationale**: Claude Desktop Chat with a local server gets no MCP skills: the Skills beta covers connectors, and `claude_desktop_config.json` servers are excluded. It does read resources and server instructions.

## R8. Claude Desktop connectivity (FR-008, SC-006, SC-011)

- **Decision**: ship a built-in stdio bridge, `fallout-tracker mcp-bridge`.
  - It is a transparent relay: it reads newline-delimited JSON-RPC messages from stdin, POSTs each one to `FALLOUT_TRACKER_MCP_URL`, and writes the responses to stdout. The default URL is `http://127.0.0.1:47823/mcp`.
  - It adds `Authorization: Bearer $FALLOUT_TRACKER_MCP_SECRET` and the protocol headers:
    - `MCP-Protocol-Version`, from the negotiated version;
    - `Mcp-Method` and `Mcp-Name`, taken from `method` and `params.name` / `params.uri`, as 2026-07-28 requires.
  - It opens neither the database nor Fyne.
  - Exit codes: 0 when stdin closes, 2 when the URL or secret is missing. Errors go to stderr. When the app is not running, the client gets a JSON-RPC error with the text "tracker is not running or the assistant connection is off".
  - Claude Desktop is configured through `claude_desktop_config.json`:
    `{"command": "<path-to-fallout-tracker>", "args": ["mcp-bridge"], "env": {"FALLOUT_TRACKER_MCP_SECRET": "…"}}`.
  - The Desktop "Code" tab runs Claude Code and connects directly (R2), as Claude Code does.
  - `mcp-remote` (npm 0.17.1) is documented as an alternative for users who already have Node.js.
- **Rationale**:
  - Claude Desktop custom connectors are reached "from Anthropic's infrastructure", so they cannot reach `127.0.0.1`.
  - Local servers in `claude_desktop_config.json` are stdio only.
  - A Go relay avoids a Node.js dependency for the GM and keeps the secret out of `args`.
  - Because confirmation falls back to the desktop dialog, the bridge does not need to translate elicitation between protocol eras.
- **Risk and spike**: relaying a legacy-era client through the stateless handler (initialize, then tools/call, with header handling) is verified by an integration test that uses the SDK client over a stdio pipe to the bridge to an httptest server. This is task-plan item P1.

## R9. Write serialization (FR-022, US6 acceptance scenario 4)

- **Decision**:
  - `app.Service` gets an operation gate built on `golang.org/x/sync/semaphore.Weighted(maxReaders)`:
    - mutating methods acquire the full weight, so they run alone;
    - read methods acquire weight 1;
    - acquisition uses the operation context (default 5 s timeout), and a timeout returns `domain.ErrTrackerBusy` ("tracker busy, try again").
  - Lock only at the exported boundary. Wrapper pairs (`CreateCampaign` → `ExecuteCreateCampaign`) lock once. Internal helpers never re-enter.
  - Add `_pragma=busy_timeout(5000)` to the SQLite DSN (`internal/store/sqlite/db.go:97,105`) as defence in depth.
  - `golang.org/x/sync` moves from an indirect to a direct dependency.
- **Rationale**:
  - Today nothing serializes operations (`internal/app/service.go:10-17`).
  - Combat operations read and then write across separate statements (`combat_action_service.go:25-35`), and `Save` replaces all combatants. Concurrent UI and assistant calls would therefore lose updates.
  - The SQLite pool is unbounded with `busy_timeout=0`, so concurrent writers fail with `SQLITE_BUSY`.
  - A context-aware weighted semaphore gives reader/writer semantics with cancellation, which `sync.RWMutex` cannot.
- **Alternatives considered**:
  - **`SetMaxOpenConns(1)` alone**: serializes statements but not read-modify-write sequences.
  - **WAL journal mode**: changes the on-disk layout (adds `-wal`/`-shm` files) and still allows lost updates.
  - **Optimistic DB versioning**: needs schema changes in every repository write path.

## R10. Live desktop refresh (FR-021, SC-004)

- **Decision**:
  - `app.Service.SubscribeChanges(func(ChangeEvent)) func()` publishes one event after each successful mutation, after the gate is released. The event carries its source (from the context, R12) and the affected IDs.
  - `fyneui` subscribes at startup and ignores desktop-sourced events, because the UI already refreshes after its own actions.
  - Assistant events are coalesced over 100 ms with `time.AfterFunc`, then run on the main goroutine through `fyne.Do(func(){ refresher.Refresh(); openEncounterList.Reload() })`.
  - Dialogs keep their widget state, because `Refresh` only repaints the main screen.
- **Rationale**:
  - The refresher currently runs only synchronously after UI actions (`main_view_refresher.go:37`).
  - `fyne.Do` is the supported way to touch widgets from another goroutine (Fyne ≥ 2.6) and is already used in `shutdown.go:22`.
  - `uiState` is unsynchronized, so every UI mutation must stay on the main goroutine.

## R11. Detecting stale desktop edits (FR-022)

- **Decision**: in-memory per-entity revisions in `app.Service`, `map[EntityKey]uint64`, guarded by the gate. Bump rules are in data-model.md.
  - Desktop editors capture the revision when they open and send `ExpectedRevision` with the save.
  - A mismatch returns `domain.ErrStaleEdit`, and a tombstone returns `domain.ErrEntityDeleted`.
  - On `ErrStaleEdit` the UI offers Reload or Overwrite. Overwrite resubmits with the fresh revision.
  - Assistant calls send 0, which skips the check.
  - Only assistant-sourced operations bump revisions (Clarifications), so the warning text "changed by the assistant" is always accurate.
- **Rationale**:
  - All writes go through one process, so in-memory revisions are exact and need no migration.
  - FR-022 is about assistant changes only. Desktop dialogs are modal, so the desktop cannot change an entity while one of its own editors is open, and bumping on desktop changes would only produce false warnings.
  - The database `updated_at` columns are too coarse. `TouchCampaign` and `UpdateCampaignResourcesByID` bump `campaigns.updated_at` on every AP or Threat change and every encounter save, which would warn falsely during combat. `domain.Encounter` has no version field at all.
- **Alternatives considered**: a `version` column on campaigns and encounters means a migration plus changes to every write query, for no benefit within a single process.

## R12. Tagging assistant actions (FR-023)

- **Decision**:
  - Add `app.WithSource(ctx, app.SourceAssistant)`, set by the adapter on every tool call. `app.SourceFromContext` returns `desktop` by default.
  - `appendOperationLog` passes the source through to `EncounterLogRepository.AppendEncounterLog`, which gets a new `source` parameter.
  - Migration `00044` adds `encounter_logs.source`. The DATA view renders `[AI]` for assistant entries.
- **Rationale**: it is the smallest change. The log messages stay the same as the desktop's (FR-023: "otherwise the same entries"), and only the source is added.

## R13. Assistant activity log (FR-024, SC-010)

- **Decision**:
  - Use the new `assistant_activity` table, with retention capped at the newest 1,000 rows on each insert.
  - The adapter records the entries, because only it knows tool names and outcomes such as declined, timed out, or refused. It calls `svc.AppendAssistantActivity`, wrapped in `runNonCriticalSideEffect`.
  - Refused requests are collapsed per reason per minute.
  - The assistant reads entries through `list_assistant_activity`. Clearing is UI-only (`svc.ClearAssistantActivity`).
- **Rationale**:
  - The log must survive restarts. FR-024 asks for at least 1,000 entries, so the newest 1,000 are kept.
  - Collapsing refusals stops a hostile local process from flooding the log.
  - Principle VII means a logging failure never fails the action itself.

## R14. Draft rosters and the difficulty preview (FR-012, FR-013)

- **Decision**:
  - Add `app.EncounterDraft{Name; PartyCharacterIDs []string; Monsters []DraftMonster{TemplateID string; Monster *domain.Combatant; Quantity int; Initiative *int}; KeepCombatantIDs []string}`.
  - Add `Service.PreviewEncounterDifficulty(ctx, EncounterDraft) (domain.EncounterDifficultyMetrics, error)` and `Service.ExecuteCreateEncounterFromDraft` / `ExecuteUpdateEncounterFromDraft`.
  - The draft expands into `[]domain.Combatant`: party members come from `ListPartyMembers`, monsters from `ListMonsterTemplates`, and each monster is repeated `Quantity` times, the same expansion as `collectCombatantsFromRows` (`input_row_collectors.go:94-110`). The result then goes through the existing `ExecuteCreateEncounter` / `ExecuteUpdateEncounter` path and `domain.EvaluateEncounterDifficulty` (`encounter.go:373`).
- **Rationale**:
  - The preview and the save must expand rosters identically (SC-002).
  - The current preview is unexported and tied to Fyne widgets (`input_rows.go:246`).
  - Putting the expansion in `internal/app` keeps it testable and shared. The desktop editor may adopt it later, but that is not required here.

## R15. Tool granularity for campaigns

- **Decision**:
  - Expose patch-style tools: `update_campaign_details`, `add_campaign_player`, `update_campaign_character`, `mark_character_available`, `mark_character_unavailable`, and `remove_campaign_player`.
  - Availability is split into two tools because MCP annotations apply to a whole tool (FR-016). Only `mark_character_unavailable` is destructive and needs confirmation.
  - Each one reads the current roster with `ListCampaignPlayers`, applies the patch, and calls the existing full-list `ExecuteUpdateCampaign` inside a single write-gate critical section (`Service.ExecuteCampaignRosterPatch`), so the read and the write cannot interleave with UI saves.
- **Rationale**:
  - Sending the full roster on every call is error-prone for an assistant. A missing player is silently deactivated (`campaign_repository.go:301-308`).
  - Patch tools make the dangerous cases explicit and confirmable:
    - `remove_campaign_player` removes a player;
    - `mark_character_unavailable` takes the character out of every encounter (`campaign_repository.go:46-117`).
  - Players are matched by lower-cased name, and renaming a character creates a new character ID. The tool descriptions say so.

## R16. Connection lifecycle

- **Decision**:
  - `mcpserver.Connection` implements `app.AssistantConnection` with `Start(ctx) error`, `Stop(ctx) error`, `Status() ConnectionStatus`, and `Apply(settings) error`.
  - Status values: `stopped`, `running`, or `failed` with a reason, such as "port 47823 is in use".
  - The UI calls `StartIfEnabled` once the main window is built, and calls `Stop` from the shutdown path (`shutdown.go`) with a 5 s `http.Server.Shutdown` deadline. In-flight requests finish or are cancelled. Pending desktop confirmations are declined.
  - A bind failure is returned to the UI and is not fatal (FR-006).
- **Rationale**: it follows the spec's lifecycle requirements (FR-002 to FR-007) using only standard-library `net/http` primitives.

## R17. Architecture placement and wiring

- **Decision**:
  - **New package `internal/mcpserver`** (adapter): it may import `internal/app`, `internal/domain`, and the go-sdk, and must not import `internal/store` or `internal/ui`.
  - **Ports live in `internal/app`**: `AssistantConnection`, `DesktopConfirmer`, `ConfirmationRequest`, `ConfirmationDecision`, `ChangeEvent`, and the repository contracts for settings and activity.
  - `fyneui` implements `DesktopConfirmer` and consumes `AssistantConnection`. `fyneui.Run` gains a `fyneui.Options{Assistant app.AssistantConnection; Confirmations *app.ConfirmationRouter}` parameter.
  - **`app.ConfirmationRouter`** is a deferred holder. It declines until the UI registers itself, which removes the construction-order cycle between mcpserver and fyneui.
  - **`cmd/fallout-tracker/main.go`**: argument `mcp-bridge` runs the bridge only. With no arguments it starts the GUI and wires the store, service, router, mcpserver, and fyneui.
- **Rationale**: the dependency direction stays `ui → app → domain` and `mcpserver → app → domain`. Neither adapter knows the other.

## R18. Constitution amendment

- **Decision**: amend the constitution to version 1.2.0 (MINOR) as the first implementation task:
  - **Principle I diagram**: add `cmd/fallout-tracker -> internal/mcpserver -> internal/app`.
  - **Principle VI**: add an exception for a loopback-only, off-by-default, token-protected assistant endpoint.
  - **Project Structure Rules**: add the `internal/mcpserver` entry.
- **Rationale**: Governance requires amendments with a rationale. Principle VI expects offline use "unless a future spec explicitly introduces networked behavior", and this spec does introduce it. Naming the exception keeps future specs from treating it as a precedent for remote access.

## R19. Out of scope (confirmed)

- **Tasks extension**: every operation takes milliseconds, and the go-sdk has no implementation.
- **File uploads**: experimental.
- **Undo**: see Clarifications.
- **Campaign deletion**: does not exist today.
- **`subscriptions/listen`**: list changes are static, so the server sends no change notifications.

## R21. Build version for `Implementation.Version` and skill metadata

- **Decision**:
  - Add `var version = "dev"` in `cmd/fallout-tracker/main.go`.
  - Set it with `-ldflags "-X main.version={{.Version}}"` in `.goreleaser.yaml` and `.goreleaser.darwin.yaml`, and with `-X main.version=$(git describe --tags --always --dirty)` in the Makefile `build` target.
  - When it is still `dev`, fall back to `debug.ReadBuildInfo().Main.Version`, unless that is `(devel)`.
  - `mcpserver.NewConnection` receives the version through its options.
- **Rationale**:
  - No version variable or ldflags exist today. The MCP `Implementation.Version` and the skills' `metadata.version` need a stable value.
  - A version change also changes the skill digests, which makes clients ask the GM to approve the updated skills (US5 acceptance scenario 3).

## R22. Classifying errors by kind

- **Decision**:
  - `internal/domain` defines `ErrInvalidInput` and `ErrNotFound`, plus `InvalidInput(err)` and `NotFound(err)`. These mark an error without changing its message.
  - `ErrEncounterNotFound` and `ErrCampaignNotFound` wrap `ErrNotFound`.
  - The application layer marks errors where they arise:
    - its own checks, through `invalidf` and `notFoundf`;
    - the validation helpers (`prepare*`, `ValidateCombatant`);
    - the results of pure domain rules (`ApplyDamage`, `Heal`, `AdvanceTurn`, resource spends).
  - The domain marks unknown combatants as not found, and the store marks its user-caused guards and duplicates.
  - The MCP adapter maps errors with `errors.Is` only. Anything unmarked is reported as `internal`.
- **Rationale**:
  - The first implementation guessed codes from words such as "must" or "not found". A new message with other wording would have been reported as `internal`, and the assistant could not correct its request.
  - Marking keeps every message unchanged, so the desktop UI is unaffected.
  - The guard test `internal/app/errors_test.go` checks that every user-caused error the Service can return is marked.
- **Alternatives considered**:
  - **A typed `FieldError{Field, Msg}` at every domain validation site** (about 84 sites): it would also fill `field`, but it is a larger change. It can still be added on top of this.
  - **Keeping the word list and adding a test**: regressions in existing messages would be caught, but new messages would not.

## R20. Known existing behavior that the contract must state

- The active encounter is the newest `encounters.updated_at` in the active campaign (`query.sql:302-308`). Every `Save` bumps that timestamp, so `update_encounter` on a non-active encounter makes it active. This matches today's desktop behavior and is kept for parity (FR-009). The tool description warns about it.
- `restart_encounter` resets only NPC HP and defeated state, and then activates the encounter (`encounter_service.go:56-87`).
- `delete_encounter` is a soft delete and writes no operation log entry today. The assistant activity log records it.
- Damage after resistance is computed as `max(amount − resistance, 0)`, or 0 when immune. Torso-only creatures always use the torso. Poison uses the global DR (`encounter.go:237-273`, `resistance.go:141,305`).
