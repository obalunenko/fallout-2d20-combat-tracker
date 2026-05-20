-- +goose Up
ALTER TABLE app_state DROP COLUMN created_at;
ALTER TABLE app_state DROP COLUMN deleted_at;

-- +goose Down
ALTER TABLE app_state ADD COLUMN created_at DATETIME NULL;
UPDATE app_state
SET created_at = COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE app_state ADD COLUMN deleted_at DATETIME NULL;
