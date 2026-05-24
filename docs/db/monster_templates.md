# monster_templates

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "monster_templates" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    name_key TEXT NOT NULL UNIQUE CHECK (trim(name_key) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
)
```

</details>

## Columns

| Name            | Type     | Default                              | Nullable | Parents                           |
| --------------- | -------- | ------------------------------------ | -------- | --------------------------------- |
| id              | TEXT     |                                      | true     |                                   |
| stat_profile_id | TEXT     |                                      | false    | [stat_profiles](stat_profiles.md) |
| name            | TEXT     |                                      | false    |                                   |
| name_key        | TEXT     |                                      | false    |                                   |
| created_at      | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                   |
| updated_at      | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                   |
| deleted_at      | DATETIME |                                      | true     |                                   |

## Constraints

| Name                                 | Type        | Definition                                                                                                     |
| ------------------------------------ | ----------- | -------------------------------------------------------------------------------------------------------------- |
| id                                   | PRIMARY KEY | PRIMARY KEY (id)                                                                                               |
| - (Foreign key ID: 0)                | FOREIGN KEY | FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles (id) ON UPDATE NO ACTION ON DELETE NO ACTION MATCH NONE |
| sqlite_autoindex_monster_templates_2 | UNIQUE      | UNIQUE (name_key)                                                                                              |
| sqlite_autoindex_monster_templates_1 | PRIMARY KEY | PRIMARY KEY (id)                                                                                               |
| -                                    | CHECK       | CHECK (trim(id) <> '')                                                                                         |
| -                                    | CHECK       | CHECK (trim(stat_profile_id) <> '')                                                                            |
| -                                    | CHECK       | CHECK (trim(name) <> '')                                                                                       |
| -                                    | CHECK       | CHECK (trim(name_key) <> '')                                                                                   |

## Indexes

| Name                                 | Definition                                                                                                 |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| idx_monster_templates_deleted_name   | CREATE INDEX idx_monster_templates_deleted_name<br />ON monster_templates(deleted_at, name COLLATE NOCASE) |
| sqlite_autoindex_monster_templates_2 | UNIQUE (name_key)                                                                                          |
| sqlite_autoindex_monster_templates_1 | PRIMARY KEY (id)                                                                                           |

## Triggers

| Name                                      | Definition                                                                                                                                                                                                                                                               |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_monster_templates_delete_stat_profile | CREATE TRIGGER trg_monster_templates_delete_stat_profile<br />AFTER DELETE ON monster_templates<br />BEGIN<br />    DELETE FROM stat_profiles<br />    WHERE id = OLD.stat_profile_id;<br />END                                                                          |
| trg_monster_templates_require_level       | CREATE TRIGGER trg_monster_templates_require_level<br />BEFORE INSERT ON monster_templates<br />WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1<br />BEGIN<br />    SELECT RAISE(ABORT, 'monster template level must be at least 1');<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
