# Assistant (MCP) connection

The tracker can let an AI assistant read and change your campaigns, encounters and live combat. It speaks the open [Model Context Protocol](https://modelcontextprotocol.io), and is tested with Claude Code and Claude Desktop.

## Safety model

- **Off by default.** Nothing listens until you turn the connection on.
- **This computer only.** The endpoint listens on `127.0.0.1`. Other computers cannot reach it, and web pages in your browser are refused.
- **Access secret.** Every request must carry the tracker's secret. You can regenerate it at any time; clients using the old one stop working immediately.
- **You confirm destructive actions.** Deleting or restarting an encounter, removing combatants, removing a player, and marking a character unavailable all ask you first: in the assistant's chat when the client supports it, otherwise in a dialog in the tracker window (50 seconds to answer).
- **Everything is logged.** Changes the assistant makes appear in the DATA view with an `[AI]` marker, and the ASSISTANT dialog keeps an activity log (the newest 1,000 entries) of changes, declined confirmations and refused requests.
- **No undo.** Fix a mistake with the opposite action (heal the damage, add back AP) or in the tracker.

## Turn the connection on

1. Click **ASSISTANT · OFF** in the header.
2. Tick **Assistant connection enabled**. The status shows `Running at http://127.0.0.1:47823/mcp`.
3. If the port is in use, the status says so. Enter another port (1024–65535) and click **APPLY**.

The choice is remembered: the connection starts with the tracker until you turn it off.

## Claude Code

Copy the **Claude Code** line from the dialog (tick **Show** first if you want to see the secret) and run it in a terminal:

```bash
claude mcp add --scope user --transport http fallout-tracker http://127.0.0.1:47823/mcp \
  --header "Authorization: Bearer <secret>"
```

Check it with `claude mcp list`. Claude Code supports everything: tools, confirmation in the terminal, and the three playbooks as skills (`/fallout-tracker:start-campaign`, `/fallout-tracker:prepare-encounter`, `/fallout-tracker:run-combat-round`). Skills need Claude Code v2.1.332 or later; on Team and Enterprise plans an admin may have to enable MCP skills.

The **Code** tab of Claude Desktop runs Claude Code and uses the same setup.

## Claude Desktop (chat)

Claude Desktop starts local servers as programs, so the tracker ships a small bridge. Copy the **Claude Desktop** snippet from the dialog into `claude_desktop_config.json` (Settings → Developer → Edit Config), then restart Claude Desktop:

```json
{
  "mcpServers": {
    "fallout-tracker": {
      "command": "/path/to/fallout-tracker",
      "args": ["mcp-bridge"],
      "env": {
        "FALLOUT_TRACKER_MCP_SECRET": "<secret>",
        "FALLOUT_TRACKER_MCP_URL": "http://127.0.0.1:47823/mcp"
      }
    }
  }
}
```

The dialog fills in the real path of the tracker binary. Typical locations:

- macOS: `/Applications/Fallout Tracker.app/Contents/MacOS/fallout-tracker`
- Linux: wherever you installed `fallout-tracker`, for example `/usr/local/bin/fallout-tracker`
- Windows: `C:\Program Files\Fallout Tracker\fallout-tracker.exe` (escape backslashes in JSON)

The tracker must be running with the connection on while you chat. In Claude Desktop chat, confirmations appear as a dialog in the **tracker window**, and the playbooks are available as resources rather than skills.

If you already use Node.js, `mcp-remote` works as an alternative bridge:

```bash
npx mcp-remote http://127.0.0.1:47823/mcp --header "Authorization:Bearer <secret>"
```

## Regenerate the secret

Click **REGENERATE SECRET** in the dialog and confirm. Paste the new secret into every client (re-run `claude mcp add`, edit `claude_desktop_config.json`).

## Troubleshooting

| Symptom | Fix |
|---|---|
| ASSISTANT shows ERROR, "port … is in use" | Another program uses the port. Choose another port and click APPLY, then update your clients. |
| Client reports 401, or "Access secret rejected" | The secret is wrong or was regenerated. Copy the current one from the dialog. |
| "Fallout tracker is not running or its assistant connection is off" | Open the tracker and turn the connection on. |
| Skills do not show in Claude Code | Update Claude Code to v2.1.332 or later; on Team/Enterprise plans ask your admin to enable MCP skills. The same guidance is available as resources. |
| A destructive action came back `timed_out` | Nobody answered the confirmation in time. Ask the assistant to try again and answer the prompt. |
