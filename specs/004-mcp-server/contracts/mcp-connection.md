# Contract: Assistant Connection (Endpoint, Security, Skills, Bridge)

## Endpoint

| Property | Value |
|---|---|
| URL | `http://127.0.0.1:<port>/mcp`. The default port is `47823`; the GM can change it (1024–65535). |
| Bind | `127.0.0.1` only. Nothing listens while the connection is off. |
| Transport | MCP Streamable HTTP, stateless (`StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true}`) |
| Methods | `POST /mcp`. `GET` and `DELETE` return 405. Any other path returns 404. |
| Protocol versions | Everything `mcp.SupportedProtocolVersions()` returns in go-sdk v1.8.0: `2026-07-28` preferred, plus `2025-11-25` and `2025-06-18` for legacy clients and the bridge |
| Request body limit | SDK default, 4 MiB |

## Request admission

The checks run in this order (research R3):

| # | Check | Failure | Activity entry |
|---|---|---|---|
| 1 | Either `Origin` is absent, or it equals exactly `http://127.0.0.1:<port>` or `http://localhost:<port>` (this endpoint's own origin). If `Sec-Fetch-Site` is present, it is `none` or `same-origin`. | 403 `{"error":"browser requests are not allowed"}` | `refused` / `browser_origin` |
| 2 | `Host` is `127.0.0.1:<port>` or `localhost:<port>` | 403 | `refused` / `bad_host` |
| 3 | `Authorization: Bearer <secret>`, compared in constant time | 401 with `WWW-Authenticate: Bearer` | `refused` / `bad_secret` |

Repeated refusals for the same reason are collapsed into one activity entry per minute. The secret never appears in responses, logs or activity entries.

## Server capabilities (returned by `server/discover` or `initialize`)

```json
{
  "tools": {},
  "resources": {},
  "extensions": { "io.modelcontextprotocol/skills": { "directoryRead": false } }
}
```

`instructions` is a short orientation text. It points to `fallout://reference/rules` and the three skills, says that destructive tools ask the GM to confirm, and warns that saving an encounter makes it active.

List and read results carry `ttlMs` and `cacheScope`, set with `ServerOptions.SetCacheable`. The content is static for the lifetime of the process.

## Skills extension (SEP-2640)

The extension is hand-rolled (research R6).

- `skills/list` returns `{skills: [{uri, frontmatter, resources: [{uri, digest: "sha256:<hex>", size}]}], resultType: "complete", ttlMs, cacheScope}`. It is not paginated: there are 3 skills.
- `skills/get {uri}` returns `{skill}` for any served skill, or the error `-32602` for an unknown URI.
- File bytes come from the standard `resources/read` of `skill://<name>/SKILL.md`. Each skill is a single file.
- **Frontmatter**:
  - `name` equals the directory segment: `start-campaign`, `prepare-encounter` or `run-combat-round`.
  - `description` is 1024 characters or fewer and says when to use the skill.
  - `metadata.version` is the tracker version.
- **Integrity**: the digests are computed at startup from the embedded bytes, so a tracker upgrade changes the digests and the client asks the GM to approve the skills again.
- **Size**: each body stays under 500 lines and roughly 5,000 tokens.

## Stdio bridge: `fallout-tracker mcp-bridge`

Used by Claude Desktop Chat (research R8).

| Item | Contract |
|---|---|
| Invocation | `<path>/fallout-tracker mcp-bridge` |
| Env | `FALLOUT_TRACKER_MCP_SECRET` (required), `FALLOUT_TRACKER_MCP_URL` (optional, default `http://127.0.0.1:47823/mcp`) |
| stdin/stdout | Newline-delimited JSON-RPC 2.0, one message per line. Notifications get no output. |
| Forwarding | Each inbound message is sent as one `POST` with `Authorization`, `Content-Type: application/json` and `Accept: application/json, text/event-stream`. Every message after `initialize` also carries `MCP-Protocol-Version`: the version in the message's `_meta`, or the one `initialize` negotiated. Messages on protocol 2026-07-28 or later add `Mcp-Method` and, for `tools/call`, `prompts/get` and `resources/read`, `Mcp-Name` (from `params.name` / `params.uri`) (research R8). |
| Cancellation | A `notifications/cancelled` from the client aborts the forwarded request with that `requestId` and is not forwarded. The tracker sees the request end, so a desktop confirmation it waited on closes as `declined`. The cancelled request gets no response. |
| HTTP timeout | Each request may wait at least 150 seconds. The bridge sets no shorter client-side timeout, so a desktop confirmation (50 s) or a slow call is never cut by the bridge. |
| Tracker unreachable | Each request gets a JSON-RPC error `-32000` with the message "Fallout tracker is not running or its assistant connection is off. Open the tracker and enable ASSISTANT." |
| HTTP 401 | JSON-RPC error `-32001` "Access secret rejected — copy the current secret from the tracker". |
| Exit codes | `0` when stdin reaches EOF. `2` when the secret is missing or the URL is invalid (message on stderr). |
| Side effects | None. It does not open the database or Fyne. |

## Client setup

The ASSISTANT dialog shows these snippets with the real port and secret, and each has a copy button.

### Claude Code (and the Claude Desktop "Code" tab)

```bash
claude mcp add --scope user --transport http fallout-tracker http://127.0.0.1:47823/mcp \
  --header "Authorization: Bearer <secret>"
```

You get the tools, MRTR elicitation (a terminal dialog) and MCP skills (`fallout-tracker:prepare-encounter` and so on). The skills need Claude Code v2.1.332 or later.

### Claude Desktop (Chat)

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "fallout-tracker": {
      "command": "/Applications/Fallout Tracker.app/Contents/MacOS/fallout-tracker",
      "args": ["mcp-bridge"],
      "env": { "FALLOUT_TRACKER_MCP_SECRET": "<secret>" }
    }
  }
}
```

You get the tools and the resources. Confirmation uses the desktop dialog, and the guidance comes from resources and server instructions, because MCP skills are not available for local Desktop servers.

`mcp-remote` is a documented alternative: `npx mcp-remote http://127.0.0.1:47823/mcp --header "Authorization:Bearer ${SECRET}"`.
