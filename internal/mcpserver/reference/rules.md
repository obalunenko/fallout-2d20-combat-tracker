# Fallout 2d20 tracker rules

This is the rules summary used by the tracker. Tools apply these rules; this text explains them.

## Encounter difficulty

- average PC level = average of the party members' levels, rounded up;
- total monster XP = sum of every monster's XP, counting each copy;
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

Without a party member the label is `Unknown`. A party with no monsters still calculates with 0 monster XP.

## Shared resources

- **Party AP** is shared by the party of the active campaign, from 0 to **6**. Adding past 6 stops at 6. Spending more than is available is rejected.
- **GM Threat** belongs to the GM, from 0 with no upper limit. Spending more than is available is rejected.

## Damage resistance (DR)

- Physical, energy and radiation DR are set **per body location**: head, torso, left arm, right arm, left leg, right leg.
- Poison DR is a single **global** value.
- Applied damage = raw damage − DR (never below 0). A combatant **immune** to a damage type takes no damage of that type.
- **Torso-only** creatures are tracked on the torso alone: every hit lands on the torso and uses torso DR.
- HP never drops below 0; at 0 a combatant is **defeated**. Healing stops at max HP.

## Turn order

Combatants act in initiative order, highest first. After the last combatant, a new round starts with the first.

## Confirmations

Deleting or restarting an encounter, removing combatants from an encounter, removing a player, and marking a character unavailable ask the GM to confirm first. A declined or timed-out confirmation changes nothing.
