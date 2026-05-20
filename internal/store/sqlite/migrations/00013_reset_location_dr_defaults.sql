-- +goose Up
-- Location columns were initially introduced as defense-per-location.
-- We now use them as DR-per-location, so reset defaults for existing rows.
UPDATE combatants
SET defense_head = 0,
    defense_torso = 0,
    defense_left_arm = 0,
    defense_right_arm = 0,
    defense_left_leg = 0,
    defense_right_leg = 0;

UPDATE player_characters
SET defense_head = 0,
    defense_torso = 0,
    defense_left_arm = 0,
    defense_right_arm = 0,
    defense_left_leg = 0,
    defense_right_leg = 0;

-- +goose Down
-- no-op
SELECT 1;
