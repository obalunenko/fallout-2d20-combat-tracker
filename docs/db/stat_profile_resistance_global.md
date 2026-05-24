# stat_profile_resistance_global

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "stat_profile_resistance_global" (
    stat_profile_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (stat_profile_id, damage_type_id),
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
)
```

</details>

## Columns

| Name            | Type     | Default                              | Nullable | Parents                           |
| --------------- | -------- | ------------------------------------ | -------- | --------------------------------- |
| stat_profile_id | TEXT     |                                      | false    | [stat_profiles](stat_profiles.md) |
| damage_type_id  | INTEGER  |                                      | false    | [damage_types](damage_types.md)   |
| resistance      | INTEGER  | 0                                    | false    |                                   |
| immune          | INTEGER  | 0                                    | false    |                                   |
| created_at      | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                   |
| updated_at      | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                   |

## Constraints

| Name                                              | Type        | Definition                                                                                                   |
| ------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------ |
| stat_profile_id                                   | PRIMARY KEY | PRIMARY KEY (stat_profile_id)                                                                                |
| damage_type_id                                    | PRIMARY KEY | PRIMARY KEY (damage_type_id)                                                                                 |
| - (Foreign key ID: 0)                             | FOREIGN KEY | FOREIGN KEY (damage_type_id) REFERENCES damage_types (id) ON UPDATE NO ACTION ON DELETE NO ACTION MATCH NONE |
| - (Foreign key ID: 1)                             | FOREIGN KEY | FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles (id) ON UPDATE NO ACTION ON DELETE CASCADE MATCH NONE |
| sqlite_autoindex_stat_profile_resistance_global_1 | PRIMARY KEY | PRIMARY KEY (stat_profile_id, damage_type_id)                                                                |
| -                                                 | CHECK       | CHECK (resistance >= 0)                                                                                      |
| -                                                 | CHECK       | CHECK (immune IN (0, 1))                                                                                     |

## Indexes

| Name                                              | Definition                                    |
| ------------------------------------------------- | --------------------------------------------- |
| sqlite_autoindex_stat_profile_resistance_global_1 | PRIMARY KEY (stat_profile_id, damage_type_id) |

## Triggers

| Name                                                  | Definition                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_stat_profile_resistance_global_poison_only_insert | CREATE TRIGGER trg_stat_profile_resistance_global_poison_only_insert<br />BEFORE INSERT ON stat_profile_resistance_global<br />WHEN NEW.resistance <> 0<br />  AND NOT EXISTS (<br />    SELECT 1 FROM damage_types dt<br />    WHERE dt.id = NEW.damage_type_id<br />      AND dt.code = 'poison'<br />  )<br />BEGIN<br />    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');<br />END                               |
| trg_stat_profile_resistance_global_poison_only_update | CREATE TRIGGER trg_stat_profile_resistance_global_poison_only_update<br />BEFORE UPDATE OF damage_type_id, resistance ON stat_profile_resistance_global<br />WHEN NEW.resistance <> 0<br />  AND NOT EXISTS (<br />    SELECT 1 FROM damage_types dt<br />    WHERE dt.id = NEW.damage_type_id<br />      AND dt.code = 'poison'<br />  )<br />BEGIN<br />    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
