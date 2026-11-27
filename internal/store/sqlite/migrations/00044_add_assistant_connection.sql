-- +goose Up
CREATE TABLE assistant_connection_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    port INTEGER NOT NULL DEFAULT 47823 CHECK (port BETWEEN 1024 AND 65535),
    access_secret TEXT NOT NULL CHECK (trim(access_secret) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE TABLE assistant_activity (
    id TEXT PRIMARY KEY,
    occurred_at DATETIME NOT NULL,
    operation TEXT NOT NULL CHECK (trim(operation) <> ''),
    entity_kind TEXT NULL CHECK (entity_kind IN (
        'campaign', 'player_character', 'encounter', 'combatant', 'monster_template'
    )),
    entity_id TEXT NULL,
    entity_label TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL CHECK (outcome IN ('done', 'rejected', 'declined', 'timed_out', 'refused')),
    detail TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE INDEX idx_assistant_activity_occurred
ON assistant_activity(occurred_at DESC);

ALTER TABLE encounter_logs
ADD COLUMN source TEXT NOT NULL DEFAULT 'desktop' CHECK (source IN ('desktop', 'assistant'));

-- +goose Down
DROP INDEX IF EXISTS idx_assistant_activity_occurred;
DROP TABLE IF EXISTS assistant_activity;
DROP TABLE IF EXISTS assistant_connection_settings;

-- encounter_logs is only a child table (nothing references it), so it can be
-- rebuilt inside the migration transaction with foreign keys left on.
DROP TABLE IF EXISTS encounter_logs_v44;

CREATE TABLE encounter_logs_v44 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    message TEXT NOT NULL CHECK (trim(message) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO encounter_logs_v44 (id, encounter_id, round, message, created_at, updated_at, deleted_at)
SELECT id, encounter_id, round, message, created_at, updated_at, deleted_at
FROM encounter_logs
ORDER BY rowid ASC;

DROP TABLE encounter_logs;
ALTER TABLE encounter_logs_v44 RENAME TO encounter_logs;

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);
