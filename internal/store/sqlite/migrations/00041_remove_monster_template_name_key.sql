-- +goose Up
DROP TRIGGER IF EXISTS trg_monster_templates_delete_stat_profile;
DROP TRIGGER IF EXISTS trg_monster_templates_require_level;
DROP TRIGGER IF EXISTS trg_stat_profiles_monster_template_level_update;
DROP INDEX IF EXISTS idx_monster_templates_deleted_name;

CREATE TABLE monster_templates_v5 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
);

INSERT INTO monster_templates_v5 (
    id, stat_profile_id, name, created_at, updated_at, deleted_at
)
SELECT
    id,
    stat_profile_id,
    name,
    created_at,
    updated_at,
    deleted_at
FROM monster_templates;

DROP TABLE monster_templates;
PRAGMA legacy_alter_table = ON;
ALTER TABLE monster_templates_v5 RENAME TO monster_templates;
PRAGMA legacy_alter_table = OFF;

CREATE UNIQUE INDEX idx_monster_templates_name_normalized
ON monster_templates(lower(trim(name)));

CREATE INDEX idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_delete_stat_profile
AFTER DELETE ON monster_templates
BEGIN
    DELETE FROM stat_profiles
    WHERE id = OLD.stat_profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_require_level
BEFORE INSERT ON monster_templates
WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profiles_monster_template_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trg_monster_templates_delete_stat_profile;
DROP TRIGGER IF EXISTS trg_monster_templates_require_level;
DROP TRIGGER IF EXISTS trg_stat_profiles_monster_template_level_update;
DROP INDEX IF EXISTS idx_monster_templates_deleted_name;
DROP INDEX IF EXISTS idx_monster_templates_name_normalized;

CREATE TABLE monster_templates_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    name_key TEXT NOT NULL UNIQUE CHECK (trim(name_key) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
);

INSERT INTO monster_templates_v4 (
    id, stat_profile_id, name, name_key, created_at, updated_at, deleted_at
)
SELECT
    id,
    stat_profile_id,
    name,
    lower(trim(name)),
    created_at,
    updated_at,
    deleted_at
FROM monster_templates;

DROP TABLE monster_templates;
PRAGMA legacy_alter_table = ON;
ALTER TABLE monster_templates_v4 RENAME TO monster_templates;
PRAGMA legacy_alter_table = OFF;

CREATE INDEX idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_delete_stat_profile
AFTER DELETE ON monster_templates
BEGIN
    DELETE FROM stat_profiles
    WHERE id = OLD.stat_profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_require_level
BEFORE INSERT ON monster_templates
WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profiles_monster_template_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd
