# Contract: Desktop UI Changes

This contract covers the Fyne screens and dialogs changed by this feature. All widget updates triggered from outside the main goroutine go through `fyne.Do` (research R10).

## Header: ASSISTANT button and status

- **Placement**: a new `ASSISTANT` button in the header (`main_screen.go:197-208`, laid out by `newMainContentWithHeader`), next to the existing campaign and encounter actions.
- **Status**: the label suffix shows `ASSISTANT · OFF`, `ASSISTANT · ON`, or `ASSISTANT · ERROR`.
- **Updates**: the label updates whenever `AssistantConnection.Status()` changes.

## Assistant connection dialog (new: `assistant_connection_dialog.go`)

| Element | Behavior |
|---|---|
| Enable switch | Toggles `enabled` and saves it, then calls `Start` or `Stop`. A failure leaves the switch on and shows the error (FR-003, FR-006). |
| Status line | `Running at http://127.0.0.1:47823/mcp`, `Stopped`, or `Failed: port 47823 is in use — choose another port` |
| Port entry + Apply | Accepts 1024–65535 and rejects other values with a field message. Apply restarts the listener if it is running. |
| Access secret | Masked by default. A Show toggle reveals it, and a Copy button copies it. |
| Regenerate secret | Asks for confirmation, then saves a new secret. Old-secret requests are refused from then on. The dialog notes that each client must be updated. |
| Setup snippets | Two read-only blocks, Claude Code and Claude Desktop, filled with the real port and secret. Each has a Copy button. |
| Activity list | The newest 200 entries, newest first: `time · operation · entity · outcome · detail`. A Clear button asks for confirmation first. The list refreshes on assistant change events while the dialog is open. |

The dialog never logs or prints the secret.

## Confirmation dialog for assistant actions (new: `assistant_confirm_dialog.go`)

- **When**: it implements `app.DesktopConfirmer` and is used only when the client cannot elicit (FR-015).
- **Content**: the title is "Assistant wants to…". The body is the confirmation text from `contracts/mcp-tools.md`, built from stored state. The buttons are `Confirm` (danger importance) and `Cancel`.
- **Queue**:
  - Requests are shown one at a time, first in first out.
  - Each request has its own 50-second deadline, measured from the moment it is enqueued. The dialog shows a countdown for the current request and a line such as "2 more requests waiting".
  - An expired request closes as `timed_out`, including one that expires while still queued: it is removed from the queue without being shown and does not run.
  - Context cancellation closes the dialog, or removes the queued request, as `declined`.
- **Visibility**: if the window is minimized, the app requests attention (`RequestFocus`), so the GM notices the dialog.

## Refresh after assistant changes (FR-021)

- **Main screen**: STAT, CAMP and DATA tabs are refreshed via `mainViewRefresher.Refresh` after each assistant-sourced `ChangeEvent`. Events are coalesced over 100 ms.
- **Encounter list dialog** (`encounter_list_dialog.go`): if open, it reloads its list and keeps the selection when the selected encounter still exists. If the selected encounter was deleted, the selection clears and a note appears in the dialog.
- **Editor dialogs** (campaign editor, campaign list → edit, encounter editor): they do **not** reload, so unsaved input is kept.

## Stale-edit handling (FR-022)

Editors capture `svc.Revision(EntityKey)` when they open, and send `ExpectedRevision` when they save.

| Service result | UI |
|---|---|
| Success | As today: the dialog closes and the main view refreshes |
| `domain.ErrStaleEdit` | A modal message: "This {encounter/campaign} was changed by the assistant after you opened it." It offers **Reload**, which discards your changes and reopens the editor with the current data, and **Overwrite**, which resubmits your values against the current revision. Nothing is saved until you choose. |
| `domain.ErrEntityDeleted` | A modal message: "This encounter was deleted by the assistant." It offers only **Close**, which closes the editor without saving. |
| `domain.ErrTrackerBusy` | An error message: "The tracker is busy, try again." The editor stays open with the input kept. |

## DATA view marker (FR-023)

`refreshEncounterDataLog` (`data_log.go:14`) renders assistant entries as `[2026-11-27 20:15:03] [R2] [AI] Damage -> Raider type:physical ...`. Desktop entries stay unchanged.

## Shutdown

The quit path (`shutdown.go`) calls `AssistantConnection.Stop` with a 5-second deadline before `app.Quit`. Pending confirmation dialogs close as `declined`.
