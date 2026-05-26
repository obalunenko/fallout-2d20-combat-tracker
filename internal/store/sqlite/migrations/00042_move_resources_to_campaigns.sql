-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS campaigns_v4;

CREATE TABLE campaigns_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    start_date DATETIME NOT NULL,
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0 AND party_ap <= 6),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

INSERT INTO campaigns_v4 (
    id, name, start_date, party_ap, gm_threat, created_at, updated_at, deleted_at
)
SELECT
    c.id,
    c.name,
    c.start_date,
    CASE
        WHEN COALESCE((
            SELECT e.party_ap
            FROM encounters e
            WHERE e.campaign_id = c.id
              AND e.deleted_at IS NULL
            ORDER BY e.updated_at DESC, e.id DESC
            LIMIT 1
        ), 0) > 6 THEN 6
        ELSE COALESCE((
            SELECT e.party_ap
            FROM encounters e
            WHERE e.campaign_id = c.id
              AND e.deleted_at IS NULL
            ORDER BY e.updated_at DESC, e.id DESC
            LIMIT 1
        ), 0)
    END,
    COALESCE((
        SELECT e.gm_threat
        FROM encounters e
        WHERE e.campaign_id = c.id
          AND e.deleted_at IS NULL
        ORDER BY e.updated_at DESC, e.id DESC
        LIMIT 1
    ), 0),
    c.created_at,
    c.updated_at,
    c.deleted_at
FROM campaigns c;

DROP TABLE campaigns;
ALTER TABLE campaigns_v4 RENAME TO campaigns;

ALTER TABLE encounters DROP COLUMN party_ap;
ALTER TABLE encounters DROP COLUMN gm_threat;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

ALTER TABLE encounters ADD COLUMN party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0);
ALTER TABLE encounters ADD COLUMN gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0);

UPDATE encounters
SET party_ap = COALESCE((
        SELECT c.party_ap
        FROM campaigns c
        WHERE c.id = encounters.campaign_id
    ), 0),
    gm_threat = COALESCE((
        SELECT c.gm_threat
        FROM campaigns c
        WHERE c.id = encounters.campaign_id
    ), 0);

DROP TABLE IF EXISTS campaigns_v4_down;

CREATE TABLE campaigns_v4_down (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    start_date DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

INSERT INTO campaigns_v4_down (id, name, start_date, created_at, updated_at, deleted_at)
SELECT id, name, start_date, created_at, updated_at, deleted_at
FROM campaigns;

DROP TABLE campaigns;
ALTER TABLE campaigns_v4_down RENAME TO campaigns;

PRAGMA foreign_keys = ON;
