---
name: prepare-encounter
description: "Build a Fallout 2d20 combat encounter for the active party at a target difficulty (Trivial, Simple, Average, Hard or Deadly) using saved monster templates. Use when the GM asks for a fight, an ambush, or a balanced encounter."
metadata:
  version: dev
---

# Prepare an encounter

## How difficulty works

The tracker computes difficulty from the party members and monsters in the roster:

- average PC level = average of the party members' levels, rounded up;
- total monster XP = the sum of every monster's XP (a monster added 3 times counts 3 times);
- XP baseline = total monster XP ÷ number of party members;
- encounter level = max(1, floor((XP baseline − 10) ÷ 10));
- difference = encounter level − average PC level.

| Difference | Label |
|---|---|
| below −2 | Trivial |
| −2 or −1 | Simple |
| 0 or 1 | Average |
| 2 to 5 | Hard |
| above 5 | Deadly |

With no party member the label is `Unknown`.

## Steps

1. **Know the party.** Call `list_party_members`. Use every member unless the GM says otherwise; keep their `character_id`.
2. **Know the monsters.** Call `list_monster_templates`. If the GM wants a creature that has no template, create it with `save_monster_template` (level, XP, HP, Defense, DR per location, poison DR, immunities, and `torso_only` for creatures tracked only on the torso).
3. **Tune without saving.** Call `preview_encounter_difficulty` with the party ids and monsters as `{template_id, quantity}`. Compare the label with the target and adjust quantities or monster choice. Raise XP to go harder, lower it to go easier. Repeat until the label matches. Previews store nothing.
4. **Save it.** Call `create_encounter` with a name, the same party ids and the same monsters. It is created in the active campaign and becomes the active encounter.
5. **Report** the name, the roster and the difficulty label to the GM.

## Changing it later

- `update_encounter` takes `keep_combatant_ids` (combatants to keep with their current HP) plus additions. Leaving a stored combatant out removes it, and the GM is asked to confirm. Saving an encounter makes it the active encounter.
- `restart_encounter` resets NPC HP and the round, after GM confirmation.
- `delete_encounter` removes it, after GM confirmation.
