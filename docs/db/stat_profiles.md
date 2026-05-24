# stat_profiles

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "stat_profiles" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 0),
    xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
    initiative INTEGER NOT NULL DEFAULT 1 CHECK (initiative >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp)
)
```

</details>

## Columns

| Name       | Type     | Default                              | Nullable | Children                                                                                                                                                                                                                                                          |
| ---------- | -------- | ------------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id         | TEXT     |                                      | true     | [stat_profile_resistance_global](stat_profile_resistance_global.md) [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) [combatants](combatants.md) [monster_templates](monster_templates.md) [player_characters](player_characters.md) |
| torso_only | INTEGER  | 0                                    | false    |                                                                                                                                                                                                                                                                   |
| level      | INTEGER  | 1                                    | false    |                                                                                                                                                                                                                                                                   |
| xp         | INTEGER  | 0                                    | false    |                                                                                                                                                                                                                                                                   |
| initiative | INTEGER  | 1                                    | false    |                                                                                                                                                                                                                                                                   |
| hp         | INTEGER  | 1                                    | false    |                                                                                                                                                                                                                                                                   |
| max_hp     | INTEGER  | 1                                    | false    |                                                                                                                                                                                                                                                                   |
| defense    | INTEGER  | 0                                    | false    |                                                                                                                                                                                                                                                                   |
| created_at | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                                                                                                                                                                                                                                                   |
| updated_at | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                                                                                                                                                                                                                                                   |
| deleted_at | DATETIME |                                      | true     |                                                                                                                                                                                                                                                                   |

## Constraints

| Name                             | Type        | Definition                   |
| -------------------------------- | ----------- | ---------------------------- |
| id                               | PRIMARY KEY | PRIMARY KEY (id)             |
| sqlite_autoindex_stat_profiles_1 | PRIMARY KEY | PRIMARY KEY (id)             |
| -                                | CHECK       | CHECK (trim(id) <> '')       |
| -                                | CHECK       | CHECK (torso_only IN (0, 1)) |
| -                                | CHECK       | CHECK (level >= 0)           |
| -                                | CHECK       | CHECK (xp >= 0)              |
| -                                | CHECK       | CHECK (initiative >= 0)      |
| -                                | CHECK       | CHECK (hp >= 0)              |
| -                                | CHECK       | CHECK (max_hp >= 1)          |
| -                                | CHECK       | CHECK (defense >= 0)         |
| -                                | CHECK       | CHECK (hp <= max_hp)         |

## Indexes

| Name                             | Definition       |
| -------------------------------- | ---------------- |
| sqlite_autoindex_stat_profiles_1 | PRIMARY KEY (id) |

## Triggers

| Name                                            | Definition                                                                                                                                                                                                                                                                                                                  |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_stat_profiles_monster_template_level_update | CREATE TRIGGER trg_stat_profiles_monster_template_level_update<br />BEFORE UPDATE OF level ON stat_profiles<br />WHEN NEW.level < 1<br />  AND EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.stat_profile_id = NEW.id)<br />BEGIN<br />    SELECT RAISE(ABORT, 'monster template level must be at least 1');<br />END |
| trg_stat_profiles_player_character_level_update | CREATE TRIGGER trg_stat_profiles_player_character_level_update<br />BEFORE UPDATE OF level ON stat_profiles<br />WHEN NEW.level < 1<br />  AND EXISTS (SELECT 1 FROM player_characters pc WHERE pc.stat_profile_id = NEW.id)<br />BEGIN<br />    SELECT RAISE(ABORT, 'player character level must be at least 1');<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
