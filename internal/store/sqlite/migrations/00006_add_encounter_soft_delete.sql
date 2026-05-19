-- +goose Up
ALTER TABLE encounters ADD COLUMN deleted_at DATETIME NULL;

CREATE INDEX IF NOT EXISTS idx_encounters_deleted_updated
ON encounters(deleted_at, updated_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_encounters_deleted_updated;
ALTER TABLE encounters DROP COLUMN deleted_at;
