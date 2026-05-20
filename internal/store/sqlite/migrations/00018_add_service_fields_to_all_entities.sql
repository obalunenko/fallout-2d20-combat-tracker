-- +goose Up
ALTER TABLE encounters ADD COLUMN created_at DATETIME NULL;
UPDATE encounters
SET created_at = COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE combatants ADD COLUMN created_at DATETIME NULL;
ALTER TABLE combatants ADD COLUMN updated_at DATETIME NULL;
ALTER TABLE combatants ADD COLUMN deleted_at DATETIME NULL;
UPDATE combatants
SET created_at = COALESCE(created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at = COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE encounter_logs ADD COLUMN updated_at DATETIME NULL;
ALTER TABLE encounter_logs ADD COLUMN deleted_at DATETIME NULL;
UPDATE encounter_logs
SET updated_at = COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE campaigns ADD COLUMN created_at DATETIME NULL;
ALTER TABLE campaigns ADD COLUMN deleted_at DATETIME NULL;
UPDATE campaigns
SET created_at = COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE app_state ADD COLUMN created_at DATETIME NULL;
ALTER TABLE app_state ADD COLUMN updated_at DATETIME NULL;
ALTER TABLE app_state ADD COLUMN deleted_at DATETIME NULL;
UPDATE app_state
SET created_at = COALESCE(created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at = COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE players ADD COLUMN created_at DATETIME NULL;
ALTER TABLE players ADD COLUMN updated_at DATETIME NULL;
ALTER TABLE players ADD COLUMN deleted_at DATETIME NULL;
UPDATE players
SET created_at = COALESCE(created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at = COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE player_characters ADD COLUMN created_at DATETIME NULL;
ALTER TABLE player_characters ADD COLUMN updated_at DATETIME NULL;
ALTER TABLE player_characters ADD COLUMN deleted_at DATETIME NULL;
UPDATE player_characters
SET created_at = COALESCE(created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at = COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

-- +goose Down
ALTER TABLE player_characters DROP COLUMN deleted_at;
ALTER TABLE player_characters DROP COLUMN updated_at;
ALTER TABLE player_characters DROP COLUMN created_at;

ALTER TABLE players DROP COLUMN deleted_at;
ALTER TABLE players DROP COLUMN updated_at;
ALTER TABLE players DROP COLUMN created_at;

ALTER TABLE app_state DROP COLUMN deleted_at;
ALTER TABLE app_state DROP COLUMN updated_at;
ALTER TABLE app_state DROP COLUMN created_at;

ALTER TABLE campaigns DROP COLUMN deleted_at;
ALTER TABLE campaigns DROP COLUMN created_at;

ALTER TABLE encounter_logs DROP COLUMN deleted_at;
ALTER TABLE encounter_logs DROP COLUMN updated_at;

ALTER TABLE combatants DROP COLUMN deleted_at;
ALTER TABLE combatants DROP COLUMN updated_at;
ALTER TABLE combatants DROP COLUMN created_at;

ALTER TABLE encounters DROP COLUMN created_at;
