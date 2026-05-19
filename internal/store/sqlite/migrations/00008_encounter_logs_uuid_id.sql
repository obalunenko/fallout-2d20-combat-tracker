-- +goose Up
PRAGMA foreign_keys = OFF;

CREATE TABLE encounter_logs_v2 (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    round INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO encounter_logs_v2 (id, encounter_id, round, message, created_at)
SELECT
    LOWER(HEX(RANDOMBLOB(4))) || '-' ||
    LOWER(HEX(RANDOMBLOB(2))) || '-' ||
    '4' || SUBSTR(LOWER(HEX(RANDOMBLOB(2))), 2) || '-' ||
    SUBSTR('89ab', 1 + (ABS(RANDOM()) % 4), 1) || SUBSTR(LOWER(HEX(RANDOMBLOB(2))), 2) || '-' ||
    LOWER(HEX(RANDOMBLOB(6))),
    encounter_id,
    round,
    message,
    created_at
FROM encounter_logs;

DROP TABLE encounter_logs;
ALTER TABLE encounter_logs_v2 RENAME TO encounter_logs;

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

CREATE TABLE encounter_logs_v1 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    encounter_id TEXT NOT NULL,
    round INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO encounter_logs_v1 (encounter_id, round, message, created_at)
SELECT encounter_id, round, message, created_at
FROM encounter_logs
ORDER BY created_at ASC, id ASC;

DROP TABLE encounter_logs;
ALTER TABLE encounter_logs_v1 RENAME TO encounter_logs;

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

PRAGMA foreign_keys = ON;
