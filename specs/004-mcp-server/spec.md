# Feature Specification: Assistant Control Of Campaigns And Encounters (MCP Server)

**Feature Branch**: Not created (the `before_specify` hooks allocate the directory only; no branch hook is configured)

**Created**: 2026-11-27

**Status**: Draft

**Input**: User description: "реализуй mcp чтобы можно было управлять энкаунтерами и кампаниями https://claude.com/resources/articles/skills-over-mcp-tasks-and-more. Если что-то непонятно - задавай вопросы" (Implement MCP so that encounters and campaigns can be managed; reference: the 2026-11-24 announcement "MCP servers can now ship skills, run tasks, and more with Claude".)

## Clarifications

### Session 2026-11-27

- Q: Should the first release include live combat actions (advance turn, damage, healing, Party AP and GM Threat), or only preparation and management? → A: Preparation and management plus live combat.
- Q: How should the assistant connection run? → A: Embedded in the running desktop app, reachable only from this computer. It is available only while the app is open.
- Q: Which capabilities from the linked article should the first release include? → A: Operations (tools), server-shipped skills (GM playbooks), and in-conversation confirmation (elicitation). Long-running tasks and file uploads are out of scope.
- Q: How should the local connection be protected? → A: Off by default. Every request must present a tracker-generated access secret, and requests from web pages are refused (FR-003, FR-005).
- Q: How should a destructive operation be confirmed when the assistant client cannot show an in-conversation confirmation? → A: The desktop app shows its own confirmation dialog. The assistant cannot bypass it by claiming the GM agreed (FR-015).
- Q: What happens when the GM has unsaved edits open in a desktop editor and the assistant changes or deletes the same entity? → A: The assistant's change is stored. When the GM saves, the editor warns that the entity changed, and the GM chooses to reload or overwrite. If the entity was deleted, the editor can only be closed (FR-022).
- Q: Should actions taken through the assistant be traceable? → A: Yes, in two places. Encounter operation-log entries are marked with the assistant as their source, and a separate assistant activity log in the desktop app records every change, its outcome, declined confirmations, and refused requests (FR-023, FR-024).
- Q: Should assistant actions be undoable in this feature? → A: No. Mistakes are corrected with opposite operations (healing, editing, AP or Threat changes) or in the desktop app, using the activity log to see what happened. Undo is a possible later feature for both the desktop app and the assistant.
- Q: Which assistant clients must the connection support and be verified with? → A: Claude Code and Claude Desktop. Both are required, documented, and covered by acceptance testing, including the bridge Claude Desktop needs to reach a local address (FR-008, SC-006, SC-011).
- Q: Can the assistant read the assistant activity log? → A: Yes, read-only. The assistant can list recent entries, for example to correct its own last action in a new conversation. It cannot clear or change the log (FR-024).
- Q: How long may the desktop confirmation dialog wait, given that many clients cut requests at 60 seconds? → A: 50 seconds for the desktop dialog. An in-conversation confirmation keeps a 2-minute window (FR-015).
- Q: What does the assistant hear when the GM does not answer a confirmation in time? → A: The operation does not run, as with a decline, and the assistant gets a distinct "timed out" result (FR-015).
- Q: Does marking a character unavailable need confirmation? → A: Yes. It removes the character from every encounter, so it is a destructive operation. It is a separate operation from making a character available, which is not destructive (FR-015, FR-016).
- Q: Which changes trigger the stale-edit warning in an open desktop editor? → A: Only changes made by the assistant (FR-022).
- Q: Which browser-related requests are refused? → A: Any request whose `Origin` is not this endpoint's own loopback address, and any cross-site browser request. Requests without an `Origin` (normal assistant clients) are allowed (FR-005).
- Q: What happens when an in-conversation confirmation comes back late or cannot be verified? → A: A genuine answer that arrives after the 2-minute window gives "timed out". A forged or reused answer, or one sent with different request details, counts as declined and is recorded as rejected (FR-015).
- Q: What happens to queued desktop confirmations that expire before they are shown? → A: They do not run and report a timeout. The open dialog shows how many more requests are waiting (FR-015).
- Q: Which activity entries does a two-step in-conversation confirmation write? → A: Exactly one, for the final outcome. The "awaiting answer" step writes none (FR-024).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Campaigns And Encounters From An Assistant (Priority: P1)

A GM connects an AI assistant (Claude Code or Claude Desktop) to the running tracker and asks questions in natural language, such as "which campaign is active?", "who is in the party?", "list my encounters and their difficulty", or "show the current round and whose turn it is". The assistant answers from the GM's real tracker data.

**Why this priority**: Read access is the smallest useful slice, carries no risk to stored data, and is required by every other story because the assistant has to find campaigns, characters, and encounters before it can change them.

**Independent Test**: With the tracker open on a database that holds two campaigns and several encounters, enable the assistant connection, connect an assistant, and ask it to list campaigns, the active campaign roster, the encounters with difficulty, one encounter's full combatant list, and that encounter's operation log. Every answer matches what the desktop app shows for the same data.

**Acceptance Scenarios**:

1. **Given** a database with several campaigns, **When** the assistant lists campaigns, **Then** each campaign is returned with its identifier, name, start date, shared Party AP and GM Threat, and whether it is the active campaign.
2. **Given** an active campaign with player characters, **When** the assistant requests the roster, **Then** each player character is returned with player name, character name, level, current and maximum HP, Defense, DR, immunities, S.P.E.C.I.A.L. values, notes, and available/unavailable status.
3. **Given** saved encounters, **When** the assistant lists encounters, **Then** each encounter is returned with its identifier, name, campaign, round, combatant count, last update time, and the same difficulty label and figures the desktop app shows (or the reason difficulty is unavailable).
4. **Given** an encounter identifier, **When** the assistant requests that encounter, **Then** it receives the round, whose turn it is, Party AP, GM Threat, and every combatant in initiative order with side, HP, Defense, DR, immunities, and defeated state.
5. **Given** an encounter with operation log entries, **When** the assistant requests the log, **Then** the entries are returned in the same order and with the same rounds and messages as the desktop app's DATA view.
6. **Given** no campaign has been created yet, **When** the assistant asks for the active campaign, **Then** it receives a clear "no campaign yet" result rather than a failure.

---

### User Story 2 - Manage Campaigns Through The Assistant (Priority: P1)

A GM asks the assistant to create a campaign with its players and characters, rename it, change its start date, add or retire players, edit a character's stats and notes, or switch the active campaign. The result is the same as doing it in the desktop CAMP screen.

**Why this priority**: Campaign setup is the first step of every session, and entering a full party by conversation is much faster than filling the form row by row.

**Independent Test**: Through the assistant only, create a campaign with four player characters, edit one character's level, HP, S.P.E.C.I.A.L., and notes, mark another unavailable (after confirming), and activate the campaign. Verify in the open desktop app that the campaign, roster, and edited values are exactly as requested.

**Acceptance Scenarios**:

1. **Given** a campaign name, a start date, and at least one player with a character, **When** the assistant creates the campaign, **Then** the campaign is stored with those players and the result includes the new campaign identifier.
2. **Given** an existing campaign, **When** the assistant updates its name, start date, or player list, **Then** the changes are stored together, and combat-relevant character changes reach the linked combatant in the active encounter by the same rule the desktop app uses.
3. **Given** a character edit that breaks a rule (for example current HP above maximum HP, or a S.P.E.C.I.A.L. value below 1), **When** the assistant submits it, **Then** the change is rejected, nothing is stored, and the error names the field and the rule so the assistant can correct the request.
4. **Given** several campaigns, **When** the assistant activates one of them, **Then** it becomes the active campaign and later encounter creation uses it.
5. **Given** a request that names a campaign or character ambiguously, **When** more than one match exists, **Then** the assistant can list the candidates with identifiers and does not guess.
6. **Given** a campaign update that would remove one or more players, or mark a character unavailable, **When** the assistant submits it, **Then** the GM is asked to confirm before the change is stored.

---

### User Story 3 - Prepare And Manage Encounters Through The Assistant (Priority: P1)

A GM asks the assistant to build an encounter, for example "three raiders and a mirelurk against the whole party", using saved monster templates and active party members. The GM can then adjust the roster, check the difficulty, activate the encounter, restart it, or delete it.

**Why this priority**: Encounter preparation is the main reason the user wants assistant control, and it depends only on the stories above.

**Independent Test**: Through the assistant, create an encounter from monster templates and party members, check that its difficulty matches the desktop calculator, change a monster quantity, activate the encounter, restart it, then delete it after confirming. After each step, verify the state in the open desktop app.

**Acceptance Scenarios**:

1. **Given** an active campaign, party members, and saved monster templates, **When** the assistant creates an encounter from chosen party members and monsters with quantities, **Then** the encounter is stored under the active campaign with combatants in initiative order, and the result includes its identifier and difficulty.
2. **Given** a draft roster, **When** the assistant asks for a difficulty preview before saving, **Then** it receives the same label, average PC level, total monster XP, encounter level, and difference that the desktop editor shows for that roster, and nothing is stored.
3. **Given** an existing encounter, **When** the assistant replaces its name or combatant roster, **Then** the encounter is updated with the same validation and party-link rules as the desktop editor.
4. **Given** an existing encounter, **When** the assistant activates it, **Then** it becomes the active encounter and an "Encounter activated" log entry is recorded.
5. **Given** an encounter, **When** the assistant restarts or deletes it, **Then** the GM is first asked to confirm, and the action happens only after the GM agrees. A declined confirmation leaves the encounter unchanged.
6. **Given** the assistant needs a new monster that has no template, **When** it saves a monster template, **Then** the template is available to later encounters in the assistant and in the desktop app.

---

### User Story 4 - Run Live Combat Through The Assistant (Priority: P2)

During play, the GM tells the assistant "next turn", "the raider takes 7 physical damage to the torso", "heal Nora for 4", or "add 2 Threat". The assistant applies the action to the active encounter by the tracker's rules and reports the result, including damage after resistance and who acts next.

**Why this priority**: Confirmed in scope (Clarifications, 2026-11-27). Live combat is the most frequent action at the table, but it needs an encounter prepared through Stories 1–3 or the desktop app first.

**Independent Test**: Activate an encounter, then through the assistant advance the turn twice, apply damage of each type to a combatant with resistances, heal a party member, and change AP and Threat. Verify that each result and the stored state match the same actions performed in the desktop app.

**Acceptance Scenarios**:

1. **Given** an active encounter, **When** the assistant advances the turn, **Then** the turn and round change as in the desktop app, and the result names the combatant whose turn it now is.
2. **Given** a combatant with DR or immunity, **When** the assistant applies damage of a type to a body location, **Then** the damage after resistance or immunity, the remaining HP, and the defeated state are computed by the same rules as the desktop app and returned to the assistant.
3. **Given** a request to spend more Party AP or GM Threat than is available, **When** the assistant submits it, **Then** it is rejected with a clear message and the resources are unchanged.
4. **Given** adding Party AP would go above the six-point cap, **When** the assistant submits it, **Then** the result reports the capped total, as the desktop app does.
5. **Given** no active encounter, **When** the assistant requests a combat action, **Then** it receives a clear "no active encounter" error and nothing changes.
6. **Given** a combat action names a combatant who is not in the active encounter, **When** the assistant submits it, **Then** it is rejected with a "combatant not found" message and nothing changes.
7. **Given** the assistant damaged the wrong combatant in an earlier conversation, **When** the GM asks it in a new conversation to fix its last action, **Then** the assistant can read the assistant activity log, identify the action, and correct it with the opposite operation. There is no undo.

---

### User Story 5 - Guided GM Workflows Shipped With The Tracker (Priority: P3)

The tracker ships step-by-step playbooks with its assistant connection: "start a new campaign", "prepare a balanced encounter for the active party", and "run a combat round". The assistant loads a playbook only when the task calls for it and follows the tracker's terms and rules, such as the difficulty bands, the six-point Party AP cap, body-location DR, and torso-only creatures, without the GM having to explain them.

**Why this priority**: The linked article presents server-shipped skills as the way to give an assistant domain knowledge. Playbooks make Stories 1–4 more reliable, but every operation still works without them.

**Independent Test**: In an assistant client that supports server-shipped skills, ask it to "prepare a hard encounter for my party". Verify that it loads the encounter-preparation playbook, uses the difficulty preview to tune the monster roster before saving, and reports the resulting difficulty label.

**Acceptance Scenarios**:

1. **Given** a client that supports server-shipped skills, **When** the GM connects the tracker, **Then** the playbooks are offered without a separate install, and their content matches the installed tracker version.
2. **Given** a client that does not support server-shipped skills, **When** the GM connects the tracker, **Then** every operation from Stories 1–4 still works, and the same guidance is available as readable reference content.
3. **Given** the tracker is upgraded, **When** the GM reconnects, **Then** the assistant receives the upgraded playbooks with no manual step.

---

### User Story 6 - Control The Connection From The Desktop App And See Changes Live (Priority: P1)

The assistant connection lives inside the desktop tracker. The GM turns it on or off from the app, sees whether it is running and the address to give an assistant, and watches changes made by the assistant appear in the app straight away. Edits made at the same moment from the app and from the assistant are never lost or corrupted.

**Why this priority**: Under the clarified runtime, the connection exists only while the app runs, so the GM needs to control it and see its state. The desktop app stays the GM's main view during play, so stale data after an assistant action would cause confusion at the table.

**Independent Test**: Start the app with the connection off and confirm that an assistant cannot connect. Turn it on, read the shown address and connect an assistant. With the STAT screen open, use the assistant to edit a character, create an encounter, activate it, and apply damage; verify that each change shows in the app without any manual refresh. Turn the connection off and confirm the assistant loses access while the app keeps working.

**Acceptance Scenarios**:

1. **Given** a fresh install, **When** the GM opens the app, **Then** the assistant connection is off and nothing outside the app can reach tracker data.
2. **Given** the connection is off, **When** the GM turns it on, **Then** the app shows that it is running, the local address, and the access details an assistant client needs. The choice is remembered after a restart.
3. **Given** the connection is on and the desktop app is open, **When** the assistant changes a campaign, an encounter, or combat state, **Then** the affected desktop views show the change within 1 second, and a dialog the GM is editing keeps its unsaved input.
4. **Given** the GM and the assistant change data at the same moment, **When** both changes are valid, **Then** both are applied one after the other with no corruption, and each side sees the final state.
5. **Given** an open desktop editor for an encounter or campaign with unsaved edits, **When** the assistant changes that entity and the GM then saves, **Then** the editor warns that the entity changed and offers Reload or Overwrite. Nothing is saved until the GM chooses.
6. **Given** an open desktop editor for an encounter, **When** the assistant deletes that encounter and the GM then saves, **Then** the editor says that the encounter was deleted and can only be closed. The encounter is not recreated.
7. **Given** the local address is already in use by another program, **When** the GM turns the connection on, **Then** the app reports the conflict, keeps working normally, and lets the GM choose another address.
8. **Given** the assistant created an encounter, damaged a combatant, had a deletion declined, and a request without the access secret was refused, **When** the GM opens the assistant activity log, **Then** all four events are listed with time, operation, entity, and outcome. The encounter's DATA view marks the creation and damage entries as coming from the assistant.
9. **Given** the connection is on, **When** the GM turns it off or closes the app, **Then** assistant operations already in progress either finish or fail cleanly with no partial change, and later requests are refused.

### Edge Cases

- The desktop app is not running, or the connection is off: the assistant client gets a "cannot connect" error. The setup documentation explains that the tracker must be open with the connection on.
- A second copy of the app is opened while the first one already serves the connection: the second copy reports that the address is in use and works without the connection.
- No campaign exists yet: read operations return an explicit empty or "not initialized" result; creating an encounter is rejected with a message that a campaign must be created first.
- No active encounter: operations on the active encounter are rejected with a "no active encounter" message.
- An identifier that does not exist, or belongs to a deleted encounter: the operation is rejected with a "not found" message and changes nothing.
- A deleted encounter is not listed and cannot be activated, restarted, or updated.
- An encounter roster with no combatants is rejected, as in the desktop editor.
- A party combatant that refers to a player character outside the encounter's campaign, or to an unavailable character, is rejected by the same rule as the desktop editor.
- Values out of range (negative amounts, current HP above maximum HP, a level or S.P.E.C.I.A.L. value below 1, an unknown damage type or body location) are rejected with field-specific messages; nothing is stored.
- Large lists: listing operations must stay usable with at least 50 campaigns, 200 encounters, and 100 monster templates.
- Text in other languages (for example Cyrillic character names and notes) is stored and returned unchanged, line breaks included.
- The GM declines a confirmation, or ignores it until it times out: the destructive operation does not run, and the assistant is told it was declined or timed out.
- The client cannot show an in-conversation confirmation: the desktop app shows the confirmation dialog instead. A request that claims the GM already agreed is still confirmed through that dialog (FR-015).
- Several destructive requests arrive while a confirmation is pending: each request gets its own confirmation, shown one at a time, and none of them runs before the GM answers it. The open dialog shows how many more requests are waiting. A queued request that reaches its deadline before it is shown does not run and reports a timeout.
- The GM answers an in-conversation confirmation after its 2-minute window: the operation does not run and the assistant is told it timed out. An answer that cannot be matched to the original request (forged, reused, or sent with different details) is treated as declined and is recorded as rejected in the activity log.
- A request comes from a web page open in a browser on the same computer, or without the access secret: it is refused and changes nothing (FR-005).
- If the operation log cannot be written after an action is saved, the action still succeeds (constitution Principle VII).

## Requirements *(mandatory)*

### Functional Requirements

**Connection and runtime**

- **FR-001**: The tracker MUST provide an assistant connection that follows the open Model Context Protocol, so that any compatible client, and not only Claude, can use it.
- **FR-002**: The assistant connection MUST run inside the desktop app and MUST be available only while the app is running and the connection is turned on. It MUST use the same data the app has open, including the data-location override.
- **FR-003**: The GM MUST be able to turn the connection on and off from the desktop app. It MUST be off by default, and the choice MUST be remembered across restarts.
- **FR-004**: While the connection is on, the desktop app MUST show its status (running, stopped, or failed with a reason), the local address to use, and the access details an assistant client needs. The GM MUST be able to copy them.
- **FR-005**: The connection MUST be reachable only from the same computer and MUST NOT need an internet connection or an online account. It MUST refuse requests that do not present the tracker's access secret, and requests that come from web pages. The GM MUST be able to replace the access secret, after which the old secret stops working.
- **FR-006**: If the connection cannot start (for example the address is in use), the app MUST report the cause, MUST keep working normally, and MUST let the GM choose another local address.
- **FR-007**: Turning the connection off, or closing the app, MUST let operations in progress finish or fail without a partial change, and MUST then refuse new requests.
- **FR-008**: The connection MUST support Claude Code and Claude Desktop, and the project MUST document how to enable the connection and connect each of them. If a client cannot reach a local address with an access secret directly (Claude Desktop is expected to need this), the documentation MUST cover the bridge it needs, and the bridge MUST still carry the access secret. Every operation, the confirmation flow (in-conversation or the desktop fallback), and the playbooks (as skills, or as reference content where skills are not available through that client) MUST work in both clients.

**Behavior parity**

- **FR-009**: Every operation available through the assistant MUST give the same stored result, validation outcome, side effects (party-link synchronization and operation-log entries), and error conditions as the same operation in the desktop app. Rules MUST NOT be reimplemented separately for the assistant connection.
- **FR-010**: Results MUST be structured (machine-readable fields with stable names) and also readable by a person, so the assistant can both reason about the data and show it. Every result MUST include the identifiers the assistant needs for follow-up operations, such as campaign, player character, encounter, combatant, and monster template identifiers.

**Campaign operations**

- **FR-011**: The assistant MUST be able to list campaigns, get the active campaign, list a campaign's player characters with all profile and combat fields, create a campaign, update a campaign's name, start date, and players (including the character profile edits from feature 003), and activate a campaign.

**Encounter operations**

- **FR-012**: The assistant MUST be able to list encounters with their difficulty summary, get one encounter in full, get the active encounter, get an encounter's operation log, list party members available for encounters, list and save monster templates, create an encounter in the active campaign, replace an encounter's name and roster, activate an encounter, restart an encounter, and delete an encounter.
- **FR-013**: The assistant MUST be able to request a difficulty preview for a draft roster without saving anything, and the preview MUST use the same calculation as the desktop encounter editor.

**Live combat**

- **FR-014**: The assistant MUST be able to advance the turn, apply damage with a damage type and body location, heal, and add or spend Party AP and GM Threat on the active encounter. Each result MUST report the outcome: damage after resistance or immunity, remaining HP, defeated state, new round and turn, and new resource totals, as applicable.

**Safety**

- **FR-015**: Destructive or hard-to-undo operations (deleting an encounter, restarting an encounter, replacing an encounter roster, a campaign update that removes players, and marking a character unavailable, which removes them from encounters) MUST ask the GM for confirmation before they run, and MUST run only if the GM accepts. The confirmation MUST appear in the assistant client's in-conversation prompt when the client supports one, and otherwise in a confirmation dialog in the desktop app. The dialog MUST name the operation and the affected entity. Request content MUST NOT be able to skip or pre-answer the confirmation. An in-conversation confirmation waits up to 2 minutes for an answer, and the desktop dialog waits up to 50 seconds. A declined or unanswered confirmation MUST leave data unchanged. The assistant MUST be told whether the GM declined or the confirmation timed out.
- **FR-016**: Every operation MUST declare whether it only reads, modifies data, or destroys data, so the assistant client can apply its own approval rules.
- **FR-017**: A failed operation MUST leave stored data unchanged and MUST return a message that says what was wrong and, where it applies, which field and which allowed range.
- **FR-025**: The kind of a failure the assistant receives (invalid input, not found, or internal) MUST be decided by what caused the error, never by the wording of its message. Every error that the GM or the assistant can cause MUST be reported as invalid input or not found, including errors added later. Only failures inside the tracker are reported as internal. (Added by BUG-001.)

**Guidance**

- **FR-018**: The connection MUST ship at least three GM playbooks as server-shipped skills: starting a new campaign, preparing a balanced encounter for the active party (using the difficulty preview), and running a combat round. The playbooks MUST be versioned and delivered with the tracker, with no separate install.
- **FR-019**: The same playbook guidance MUST also be available as readable reference content for clients that do not support server-shipped skills.
- **FR-020**: Operation descriptions MUST explain the Fallout 2d20 terms they use (Party AP, GM Threat, DR by body location, immunity, difficulty bands, torso-only creatures), so an assistant without a playbook can still use them correctly.

**Desktop consistency**

- **FR-021**: After an assistant operation changes data, the affected desktop views MUST show the change without a manual refresh, with a target of 1 second (measured by SC-004), and MUST NOT discard input the GM is entering in an open dialog.
- **FR-022**: Changes from the desktop app and from the assistant MUST be applied one at a time so neither corrupts the other. An open desktop editor MUST NOT block assistant operations. When the GM saves an editor whose entity was changed by the assistant after the editor opened, the app MUST warn and let the GM choose to reload the current state (discarding the editor's changes) or overwrite it with the editor's values. When the entity was deleted, the editor MUST say so and MUST only allow closing, without recreating the entity.

**Traceability**

- **FR-023**: Encounter operation-log entries written for assistant operations MUST be marked with the assistant as their source, so the DATA view tells them apart from actions taken in the desktop app. They are otherwise the same entries the desktop app writes.
- **FR-024**: The desktop app MUST keep an assistant activity log and show it in the assistant connection control, newest first. The log MUST record:
  - every modifying or destructive assistant operation, with time, operation, affected entity, and outcome (done, rejected with reason, declined, or timed out);
  - every refused request that reaches the tracker (missing or wrong access secret, a web-page origin, or a wrong host address), with time and reason. While the connection is off, nothing listens, so there is nothing to record. SC-008 covers that case.

  Read-only operations are not recorded. The log MUST survive restarts and keep at least the latest 1,000 entries, and the GM MUST be able to clear it. Entries MUST NOT contain the access secret. A failure to write an activity entry MUST NOT fail the operation itself (constitution Principle VII).

  The assistant MUST be able to list the most recent activity entries (newest first, with a count limit) through a read-only operation. It MUST NOT be able to clear or change them.

**Bugfix**: 2026-11-27 — BUG-001 Added FR-025: error kinds come from the cause of the error, not from its message wording.

### Fallout 2d20 Rules Impact

- No combat rule changes. Damage, resistance, immunity, healing, HP normalization, initiative ordering, turn progression, the six-point Party AP cap, GM Threat, and difficulty bands keep their current definitions, and the assistant connection reuses them.

### Data & Persistence Impact

- No new durable combat data is expected.
- **New durable data**: the assistant activity log (FR-024) and the source of each encounter log entry (FR-023). If they are stored in the tracker database, the plan must name the migration, the sqlc schema and query updates, the dbgen regeneration, and the `docs/db` regeneration. Existing log entries default to the desktop app as their source.
- The connection settings (on/off, local address, access secret) are new durable app settings. Planning must decide whether they live in the app's preferences or in the tracker database. If they go in the database, or if the conflict detection for FR-022 needs a change marker, the plan must name the migration, the sqlc schema and query updates, the dbgen regeneration, and the `docs/db` regeneration.
- Writes from the desktop app and from the assistant share one process and one data store. Planning must confirm that they are serialized as FR-022 requires.

### UI Impact

- **New**: an "Assistant connection" control reachable from the main screen header. It provides an on/off switch, a status display, the local address, copy actions for the connection details, address change, and access-secret regeneration (FR-003 to FR-006).
- **New**: an assistant activity list inside the connection control, newest first, with a Clear action (FR-024).
- **Changed**: the DATA view marks operation-log entries that come from the assistant (FR-023).
- **New**: a confirmation dialog for destructive assistant operations, used when the assistant client cannot show its own confirmation. It names the operation and the affected entity and offers Confirm and Cancel (FR-015).
- **Refresh**: the main screen (STAT, CAMP, and DATA tabs) and the encounter list dialog must refresh after assistant changes (FR-021). Open campaign, character, and encounter editor dialogs keep unsaved input. On save they show a Reload / Overwrite warning when the assistant changed their entity, and a close-only notice when it was deleted (FR-022).
- No other new screens are required.

### Key Entities

- **Assistant Connection**: The tracker's Model Context Protocol endpoint inside the desktop app. It has an on/off state, a local address, an access secret, and a status. It offers operations, playbooks, and reference content over the app's data.
- **Operation (tool)**: A named action the assistant can invoke. It has a description, input fields, a result, and a declared effect: read-only, modifying, or destructive.
- **Playbook (skill)**: Step-by-step guidance shipped with the tracker for a GM workflow (start a campaign, prepare an encounter, run a round). It is versioned with the tracker.
- **Confirmation Request**: A question shown to the GM before a destructive operation runs, in the assistant client or, as a fallback, in the desktop app. Its outcome (accepted, declined, or timed out) decides whether the operation proceeds.
- **Assistant Activity Entry**: One record in the assistant activity log. It holds the time, the operation, the affected entity (if any), and the outcome: done, rejected with reason, declined, timed out, or refused access. It is kept across restarts. The GM sees it in the desktop app; the assistant can read it but not change it.
- **Campaign, Player Character, Encounter, Combatant, Monster Template, Encounter Log Entry**: Existing tracker entities. An encounter log entry gains a source (desktop app or assistant); their meaning is otherwise unchanged (see specs 001–003).
- **Character availability (term)**: a player character is either *available* or *unavailable*. The tracker stores this as the availability status `inactive`. It is separate from the tracker's internal "active" flag, which marks the current version of a character record. This spec, the tools and the tasks use only available/unavailable.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Using only the assistant, a GM can create a campaign with four player characters and a saved encounter with a difficulty label in under 5 minutes.
- **SC-002**: For every operation offered through the assistant, a parity check against the same desktop action gives identical stored data and identical validation outcomes in 100% of the tested cases.
- **SC-003**: No destructive operation runs without the GM's confirmation (0 occurrences across the acceptance tests for FR-015).
- **SC-004**: Changes made through the assistant appear in the open desktop app within 1 second in at least 95% of trials, and no unsaved dialog input is lost in any trial.
- **SC-005**: 100% of invalid requests in the acceptance tests are rejected with a message that names the problem field, and leave stored data unchanged.
- **SC-006**: A GM who follows the setup documentation can turn the connection on and connect Claude Code in under 5 minutes, or Claude Desktop (including its bridge) in under 10 minutes.
- **SC-007**: Read operations on a database with 50 campaigns, 200 encounters, and 100 monster templates return in under 1 second.
- **SC-008**: In access tests, 0 requests succeed while the connection is off, from another computer, from a web page, or without the current access secret.
- **SC-009**: In a client that supports server-shipped skills, the encounter-preparation playbook leads the assistant to produce an encounter with the requested difficulty label in at least 4 of 5 trials.
- **SC-010**: 100% of modifying and destructive assistant operations, declined confirmations, and refused requests in the acceptance tests appear in the assistant activity log with the correct outcome.
- **SC-011**: The acceptance scenarios of User Stories 1–6 pass in both Claude Code and Claude Desktop.

## Assumptions

- The GM is the only user of the computer that runs the tracker. Multi-user access, remote access from other devices, and permission roles are out of scope.
- The GM pastes the address and access secret into the assistant client once. They need to do it again only after regenerating the secret or changing the address.
- "Manage campaigns and encounters" means the capabilities the desktop app already has, plus the read-only difficulty preview. Deleting a campaign does not exist in the tracker today and is out of scope for this feature.
- All tracker operations finish quickly, so long-running background work ("Tasks" in the article) brings no user value in this release. File uploads (an experimental capability in the article) are also out of scope; importing characters or monsters from documents can be a later feature.
- There is no undo for assistant actions in this feature (Clarifications, 2026-11-27). Wrong combat actions are corrected with opposite operations or in the desktop app.
- The assistant chooses entities by identifier. Name-based lookup is handled by listing first and then using the returned identifier.
- Operation names, descriptions, and playbooks are written in English, consistent with the rest of the project. The assistant may talk to the GM in any language.
- Clients that do not support skills or in-conversation confirmation still get full functionality: confirmation falls back to the desktop app (FR-015), and guidance to reference content and operation descriptions (FR-019, FR-020).
- Only one running copy of the app serves the connection at a time.

## Constitution Check

- **Affected Layers**: The assistant connection is a new adapter at the same level as the Fyne UI. It calls `internal/app` services and must not reimplement domain rules or touch storage directly. `cmd/fallout-tracker` assembles it and runs it with the Fyne UI in one process. The Fyne UI gains the connection control and refreshes after assistant changes. A difficulty-preview operation for draft rosters may need an application-level entry point if one does not exist yet. A change notification from `internal/app` to the UI must not make `internal/app` depend on Fyne.
- **Domain Rules**: No new domain rules. Existing validation and combat rules are reused unchanged (Principle II).
- **Persistence Impact**: No new combat data. The assistant activity log, the encounter-log source, the connection settings, and any change marker for conflict detection must follow the migration, sqlc, dbgen, and docs workflow if they are stored in the tracker database (Principles III and V).
- **Local Operation (Principle VI)**: This spec explicitly introduces networked behavior, limited to a local endpoint that only this computer can reach. It is off by default and protected by an access secret. Offline play is kept: no internet access, remote hosting, or accounts. The plan must record whether the constitution text should be amended to name this exception.
- **Audit Side Effects (Principle VII)**: Operations from the assistant write the same operation-log entries as the desktop app, marked with the assistant as their source, plus an assistant activity entry. A failed write to either log does not fail the saved action.
- **Test Impact**: Planning must include application parity tests for every exposed operation; adapter tests for input validation, result structure, effect declarations, the confirmation flow (including the desktop fallback dialog), access control, and activity logging; tests for writes from the UI and the assistant at the same time; and UI tests for the connection control and for refresh and conflict warnings after assistant changes (Principle IV).
- **New Dependency**: A Model Context Protocol library may be added to the application module. Planning must justify the choice. Tool dependencies stay in their own `tools/<tool>` modules.
- **Layering Result**: Pass for specification. No architecture exception is requested beyond the explicit, limited networked behavior under Principle VI.
