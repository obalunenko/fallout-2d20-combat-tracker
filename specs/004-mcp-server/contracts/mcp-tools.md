# Contract: MCP Tools

The server name is `fallout-tracker`, and the title is "Fallout 2d20 Combat Tracker".

All tools follow these common rules:

- **Context**: every call runs with `app.SourceAssistant` in its context (research R12).
- **Annotations**: every tool sets all four hints explicitly (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint: false`), because the SDK defaults `destructiveHint` and `openWorldHint` to `true`.
- **Inputs**: JSON Schema is inferred from Go structs. Enums are declared, integers carry `minimum` where the domain has one, and identifiers are non-empty strings.
- **Outputs**: `structuredContent` holds the DTO, and `content[0]` holds a short human-readable summary (FR-010).
- **Activity log**: each modifying or destructive operation writes exactly one activity entry, for its final outcome. When the operation is confirmed in conversation, the first call (`input_required`) writes nothing. The retry writes the entry. If the client never retries, no entry is written. Read-only tools write none (FR-024).

## Error contract (FR-017)

Failures return `isError: true` with:

- a single text content block holding the JSON object `{"error": {"code": "<code>", "message": "<text>", "field": "<field>"}}`. It is not sent as `structuredContent`, because clients that check `structuredContent` against the tool's output schema would reject it. `field` is optional. It is set only when the adapter itself knows the field: a schema violation, or a check the adapter makes such as `quantity`, `delta`, `limit`, or exactly one of `template_id` / `monster`. Errors from domain validation leave it out, because their message already names the field (SC-005).

| `code` | Cause |
|---|---|
| `invalid_input` | Schema violation, an adapter check, or an error marked `domain.ErrInvalidInput` by the domain, application or store layer. The message passes through verbatim and already names the field. |
| `not_found` | An error marked `domain.ErrNotFound`: `domain.ErrEncounterNotFound`, `domain.ErrCampaignNotFound`, an unknown combatant, character or template ID, or a deleted encounter |
| `no_campaign` | `domain.ErrCampaignNotInitialized` |
| `no_active_encounter` | `domain.ErrEncounterNotInitialized` |
| `busy` | `domain.ErrTrackerBusy` (the operation gate timed out; retry) |
| `declined` | The GM declined the confirmation, a client cancel closed it, or the confirmation answer could not be verified (forged or reused token, or different arguments; activity outcome `rejected`) |
| `timed_out` | The confirmation got no answer in time: 2 minutes for an in-conversation confirmation, 50 seconds for the desktop dialog. The operation did not run, and the assistant may offer to try again. |
| `internal` | Any unmarked error. Codes are chosen by error kind (`errors.Is`), never by message wording. The message is generic, and the detail is logged locally. |

A failed call changes no stored data. An activity entry is still written, with the outcome `rejected`, `declined` or `timed_out`.

## Shared DTOs

```text
DR         = { head, torso, left_arm, right_arm, left_leg, right_leg : int>=0 }
Resistance = { physical: DR, energy: DR, radiation: DR, poison: int>=0 }
Immunity   = { physical, energy, radiation, poison : bool }
Special    = { strength, perception, endurance, charisma, intelligence, agility, luck : int>=1 }

Campaign   = { campaign_id, name, start_date (YYYY-MM-DD), party_ap (0..6), gm_threat (>=0), is_active, updated_at }
Character  = { character_id, player_name, name, level>=1, initiative, hp>=0, max_hp>=1, defense>=0,
               resistance: Resistance, immunity: Immunity, special: Special, notes, available: bool }
Combatant  = { combatant_id, character_id (party only, else ""), name, side ("party"|"npc"), torso_only,
               level, xp, initiative, hp, max_hp, defense, resistance, immunity, defeated, is_current_turn }
Difficulty = { label ("Trivial"|"Simple"|"Average"|"Hard"|"Deadly"|"Unknown"), unavailable_reason,
               party_count, average_pc_level, total_monster_xp, xp_baseline, encounter_level, difference }
EncounterSummary = { encounter_id, campaign_id, name, round, combatant_count, updated_at, difficulty: Difficulty }
Encounter  = { encounter_id, campaign_id, name, round, turn_index, current_combatant_id,
               party_ap, gm_threat, combatants: [Combatant], difficulty: Difficulty }
MonsterTemplate = Combatant fields minus combatant_id/character_id/defeated/is_current_turn, plus template_id
LogEntry   = { round, message, created_at, source ("desktop"|"assistant") }
Activity   = { activity_id, occurred_at, operation, entity_kind, entity_id, entity_label, outcome, detail }

CharacterInput = Character minus character_id/available (all combat fields required on create; optional on patch)
MonsterInput   = { name, level>=1, xp>=0, initiative, hp>=0, max_hp>=1, defense>=0, torso_only,
                   resistance, immunity }
DraftMonster   = { template_id? | monster?: MonsterInput, quantity: int 1..50, initiative?: int }
                 (exactly one of template_id / monster)
```

## Read-only tools

`readOnlyHint: true, destructiveHint: false, idempotentHint: true`

| Tool | Input | Output | Notes |
|---|---|---|---|
| `list_campaigns` | `{}` | `{campaigns: [Campaign]}` | |
| `get_active_campaign` | `{}` | `{initialized: bool, campaign: Campaign \| null}` | No campaign yet → `initialized: false`, not an error (US1 acceptance scenario 6) |
| `list_campaign_characters` | `{campaign_id?: string}` | `{campaign_id, characters: [Character]}` | Defaults to the active campaign |
| `list_party_members` | `{}` | `{members: [Combatant]}` | Active campaign, available characters only (`ListPartyMembers`) |
| `list_encounters` | `{}` | `{encounters: [EncounterSummary]}` | Same set and difficulty as the desktop encounter list |
| `get_encounter` | `{encounter_id}` | `Encounter` | |
| `get_active_encounter` | `{}` | `{encounter: Encounter \| null}` | |
| `get_encounter_log` | `{encounter_id, limit?: 1..500 = 100}` | `{entries: [LogEntry]}` | Newest first, as stored |
| `list_monster_templates` | `{}` | `{templates: [MonsterTemplate]}` | |
| `preview_encounter_difficulty` | `{party_character_ids: [string], monsters: [DraftMonster]}` | `Difficulty` | Stores nothing. Uses the same expansion as `create_encounter` (research R14) |
| `list_assistant_activity` | `{limit?: 1..200 = 20}` | `{entries: [Activity]}` | Newest first. Read-only access to the log (FR-024) |

## Modifying tools

`readOnlyHint: false, destructiveHint: false`

| Tool | Input | Output | idempotent | Notes |
|---|---|---|---|---|
| `create_campaign` | `{name, start_date, players: [{player_name, character: CharacterInput, notes?, special?, available? = true}] (min 1)}` | `Campaign` + `characters` | no | |
| `update_campaign_details` | `{campaign_id, name?, start_date?}` | `Campaign` | yes | Roster unchanged |
| `add_campaign_player` | `{campaign_id, player_name, character: CharacterInput, notes?, special?}` | `{campaign, character: Character}` | no | Rejects a duplicate player name (case-insensitive) |
| `update_campaign_character` | `{campaign_id, character_id, patch: {name?, level?, initiative?, hp?, max_hp?, defense?, resistance?, immunity?, special?, notes?}}` | `{character: Character, replaced_character_id?: string}` | yes | Renaming creates a new `character_id` (existing repository behavior); the old ID is returned in `replaced_character_id`. Combat-relevant changes sync to the linked combatant in the active encounter (spec 003). |
| `mark_character_available` | `{campaign_id, character_id}` | `{character}` | yes | Makes an unavailable character available again. The character does not rejoin past encounters. |
| `activate_campaign` | `{campaign_id}` | `Campaign` | yes | |
| `create_encounter` | `{name, party_character_ids: [string], monsters: [DraftMonster]}` (at least one combatant) | `Encounter` | no | Created in the active campaign. Starting resources are copied from the campaign. |
| `activate_encounter` | `{encounter_id}` | `Encounter` | yes | Logs "Encounter activated" with the `assistant` source |
| `save_monster_template` | `{monster: MonsterInput}` | `{template: MonsterTemplate, replaced_existing: bool}` | yes | Upserts by case-insensitive name (existing behavior) |
| `advance_turn` | `{}` | `{encounter: Encounter, round, current_combatant: Combatant}` | no | Active encounter |
| `apply_damage` | `{combatant_id, damage_type: "physical"\|"energy"\|"radiation"\|"poison", location: "head"\|"torso"\|"left_arm"\|"right_arm"\|"left_leg"\|"right_leg", amount: int>=0}` | `{combatant: Combatant, raw: int, applied: int, defeated: bool}` | no | Torso-only creatures always take damage on the torso, and poison uses the global DR (stated in the description). |
| `heal` | `{combatant_id, amount: int>=0}` | `{combatant: Combatant, healed: int}` | no | Healing stops at max HP |
| `change_party_ap` | `{delta: int != 0}` | `{party_ap, capped: bool}` | no | A positive delta adds AP, up to a cap of 6. A negative delta spends AP and is rejected when not enough is available. |
| `change_gm_threat` | `{delta: int != 0}` | `{gm_threat}` | no | A positive delta adds Threat. A negative delta spends it and is rejected when not enough is available. |

## Destructive tools (confirmation required, FR-015)

`readOnlyHint: false, destructiveHint: true, idempotentHint: false`

| Tool | Input | When it asks for confirmation | Confirmation text (built from stored state) |
|---|---|---|---|
| `delete_encounter` | `{encounter_id}` | Always | "Delete encounter «{name}» (round {round}, {n} combatants)?" |
| `restart_encounter` | `{encounter_id}` | Always | "Restart encounter «{name}»? NPC HP resets to max and the round goes back to 1." |
| `update_encounter` | `{encounter_id, name?, keep_combatant_ids: [string], add_party_character_ids?: [string], add_monsters?: [DraftMonster]}` | When the stored combatants are not all listed in `keep_combatant_ids` | "Update encounter «{name}»: remove {k} combatant(s) ({names})?" |
| `remove_campaign_player` | `{campaign_id, character_id}` | Always | "Remove {player} ({character}) from campaign «{name}»? They are also removed from its encounters." |
| `mark_character_unavailable` | `{campaign_id, character_id}` | Always | "Mark {character} unavailable? They are removed from all encounters of «{campaign}»." |

- `update_encounter` without removals (only a rename or additions) runs without confirmation. The tool's annotation still says destructive, so the client applies its own approval rules (FR-016).
- `update_encounter` checks the roster again after the GM confirms. When the stored combatants changed meanwhile, so that `keep_combatant_ids` no longer removes exactly the confirmed combatants, nothing is saved and the result is `invalid_input` on `keep_combatant_ids` (FR-015, SC-003).
- The description of `update_encounter` warns: "Saving an encounter makes it the active encounter (tracker behavior)." See research R20.
- Combatants named in `keep_combatant_ids` keep their stored state (HP, defeated). New entries are expanded as in `create_encounter`.

### Confirmation flow

See research R5.

1. **Client supports form elicitation on protocol 2026-07-28**: the first call returns `resultType: "input_required"` with `inputRequests.confirm` and a `requestState` token. The schema is `{type: object, properties: {confirm: {type: boolean, title: "Confirm"}}, required: [confirm]}`. The retry must carry the same arguments, a valid single-use token, and `action: "accept"` with `confirm: true`. Other outcomes:
   - an authentic token used after its 2-minute expiry returns `timed_out` and the operation does not run;
   - a forged token, a reused token, or a retry with arguments that differ from those the token was issued for returns `declined`, with activity outcome `rejected` and detail "confirmation could not be verified";
   - `action: "decline"` or `"cancel"`, or `confirm: false`, returns `declined`.
2. **Otherwise**: the call blocks while the desktop app shows the confirmation dialog, for up to **50 seconds**. That stays under the 60-second request timeout common in MCP clients. The outcome is accepted, `declined` or `timed_out`. If the client cuts the request first, or the GM stops the connection, the outcome is `declined` and the operation does not run.
3. **After acceptance**: the operation runs through the normal service path. If the entity disappeared meanwhile, the result is `not_found`.

## Resources

All resources are read-only.

| URI | MIME | Content |
|---|---|---|
| `skill://start-campaign/SKILL.md` | `text/markdown` | Playbook for setting up a campaign and party |
| `skill://prepare-encounter/SKILL.md` | `text/markdown` | Playbook for building an encounter to a target difficulty with `preview_encounter_difficulty` |
| `skill://run-combat-round/SKILL.md` | `text/markdown` | Playbook for running a round: turn order, damage types and locations, AP and Threat, defeated combatants, correcting mistakes from `list_assistant_activity` |
| `fallout://reference/rules` | `text/markdown` | Rules summary: difficulty formula and bands, Party AP cap, DR model, immunity, torso-only creatures |

These resources are listed in `resources/list` for every client (FR-019). The three `skill://` resources are also published as skills (see `mcp-connection.md`).
