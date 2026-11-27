---
name: start-campaign
description: "Set up a new Fallout 2d20 campaign and its party in the GM's combat tracker. Use when the GM wants to create a campaign, enter players and their characters, or make a campaign the active one."
metadata:
  version: dev
---

# Start a campaign

The tracker stores campaigns, players and their characters. Encounters always belong to the **active** campaign, so set the campaign up before preparing fights.

## Steps

1. **Check what exists.** Call `list_campaigns`. If the GM names an existing campaign, use its `campaign_id` and skip to step 5.
2. **Collect the party.** For each player ask for:
   - player name and character name;
   - level (1 or more), initiative, current and maximum HP (current never above maximum), Defense (0 or more);
   - DR (damage resistance): physical, energy and radiation DR **per body location** (head, torso, left/right arm, left/right leg), and a single **global** poison DR. Use 0 when unknown;
   - immunities to physical, energy, radiation or poison damage (rare);
   - optional notes, and S.P.E.C.I.A.L. (Strength, Perception, Endurance, Charisma, Intelligence, Agility, Luck), each 1 or more. Omit S.P.E.C.I.A.L. to default every attribute to 1.
3. **Create it.** Call `create_campaign` with the name, a start date as `YYYY-MM-DD`, and the players. If it returns `invalid_input`, the message names the field; fix that value and call again.
4. **Confirm with the GM.** Read back the roster from `list_campaign_characters`.
5. **Activate it.** Call `activate_campaign` so new encounters use this campaign.

## Later changes

- Rename or redate: `update_campaign_details`.
- New player: `add_campaign_player`.
- Level up, HP, Defense, DR, S.P.E.C.I.A.L. or notes: `update_campaign_character`. Renaming a character gives it a **new** `character_id`; use the returned one afterwards.
- A character sits a session out: `mark_character_unavailable` removes them from every encounter of the campaign and asks the GM to confirm. `mark_character_available` brings them back for future encounters.
- A player leaves: `remove_campaign_player` asks the GM to confirm.

Never claim the GM agreed to a confirmation. If a tool returns `declined` or `timed_out`, nothing changed; ask the GM what to do.
