---
name: run-combat-round
description: "Run live Fallout 2d20 combat in the GM's tracker: turn order, damage by type and body location, healing, Party AP and GM Threat, defeated combatants, and correcting mistakes. Use during play when the GM narrates attacks, turns or resource changes."
metadata:
  version: dev
---

# Run a combat round

Combat actions always apply to the **active encounter**. Start with `get_active_encounter`; if it is null, ask the GM which encounter to start and call `activate_encounter`.

## Turn order

- Combatants act in initiative order, highest first. `is_current_turn` marks whose turn it is.
- `advance_turn` moves to the next combatant; after the last one a new round starts. It returns who acts now. Announce it.

## Damage

Call `apply_damage` with `combatant_id`, `damage_type`, `location` and the raw `amount` the GM rolled.

- Damage types: `physical`, `energy`, `radiation`, `poison`.
- Locations: `head`, `torso`, `left_arm`, `right_arm`, `left_leg`, `right_leg`.
- Applied damage = raw − DR at that location for physical, energy and radiation. Poison uses the global poison DR. An immune combatant takes 0.
- Torso-only creatures always take the hit on the torso, whatever location you send.
- HP never goes below 0. At 0 the combatant is **defeated**. Report the applied damage and remaining HP.

Use `heal` to restore HP; healing stops at max HP.

## Resources

- `change_party_ap` with a positive delta adds Party AP (capped at 6; `capped: true` tells you the cap cut it) and with a negative delta spends it. Spending more than is available is rejected and changes nothing.
- `change_gm_threat` does the same for GM Threat, which has no cap.

## Mistakes

There is no undo. To fix a wrong call:

1. Call `list_assistant_activity` to see the last actions you took, with their amounts and targets.
2. Apply the opposite operation: `heal` the damage you applied, add back spent AP or Threat, or ask the GM to correct turn order in the tracker.

## Ending a fight

- `restart_encounter` resets NPC HP and the round; `delete_encounter` removes the encounter. Both ask the GM to confirm. Never claim the GM agreed; if the result is `declined` or `timed_out`, nothing changed.
