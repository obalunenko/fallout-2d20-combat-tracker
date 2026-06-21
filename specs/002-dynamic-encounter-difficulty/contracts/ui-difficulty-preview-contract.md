# UI Contract: Dynamic Encounter Difficulty Preview

## Surface

The encounter create/edit dialog exposes a difficulty preview area. The preview reads only the current dialog draft state and does not save encounter data.

## Inputs

- Party rows currently present in the encounter editor.
- Monster/NPC rows currently present in the encounter editor.
- Party level values.
- Monster XP values.
- Monster quantity values.
- Row side changes between party and NPC where the editor permits side changes.
- Rows loaded from the campaign party library.
- Rows loaded from the monster library.
- Row add/remove actions.

## Output

When calculation input is valid, the preview displays:

- Difficulty label: Trivial, Simple, Average, Hard, or Deadly.
- Supporting metrics sufficient for GM verification: party count, rounded average PC level, total monster XP, XP baseline, encounter level, and difference.

When calculation input is unavailable or invalid, the preview displays:

- Difficulty: Unknown or equivalent unavailable state.
- A concise reason such as missing players or invalid numeric draft input.

## Refresh Contract

The preview MUST refresh after:

- player level changes;
- monster XP changes;
- monster quantity changes;
- row side changes;
- row add;
- row remove;
- party library load;
- monster library load.

The refresh MUST use the unsaved draft values currently visible in the dialog.

## Save/Cancel Contract

- Save persists the encounter through the existing save flow.
- Cancel closes the editor without persisting draft changes.
- Preview refreshes MUST NOT create, update, or delete encounters.
- Preview refreshes MUST NOT require a database write.

## Calculation Contract

1. Average PC Level = `ceil(sum(selected player levels) / selected player count)`.
2. Total Monster XP = `sum(monster XP * monster quantity)`.
3. XP Baseline = `Total Monster XP / selected player count`.
4. Encounter Level = `floor((XP Baseline - 10) / 10)`, minimum 1.
5. Difference = `Encounter Level - Average PC Level`.
6. Label:
   - `< -2`: Trivial
   - `-2` or `-1`: Simple
   - `0` or `1`: Average
   - `2` through `5`: Hard
   - `> 5`: Deadly

## Error Handling

- Missing selected players: unavailable preview.
- Invalid party level: unavailable preview.
- Invalid monster XP: unavailable preview.
- Invalid monster quantity: unavailable preview.
- Missing monsters with valid players: calculate with Total Monster XP of 0 and minimum Encounter Level of 1.
