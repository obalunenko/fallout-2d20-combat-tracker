-- +goose Up
CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_encounters_campaign_deleted_updated;
