# combatants

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "combatants" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    player_character_id TEXT NULL,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    side TEXT NOT NULL CHECK (side IN ('party', 'npc')),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
    defeated INTEGER NOT NULL DEFAULT 0 CHECK (defeated IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id)
)
```

</details>

## Columns

| Name                | Type     | Default                              | Nullable | Parents                                   |
| ------------------- | -------- | ------------------------------------ | -------- | ----------------------------------------- |
| id                  | TEXT     |                                      | true     |                                           |
| encounter_id        | TEXT     |                                      | false    | [encounters](encounters.md)               |
| stat_profile_id     | TEXT     |                                      | false    | [stat_profiles](stat_profiles.md)         |
| player_character_id | TEXT     |                                      | true     | [player_characters](player_characters.md) |
| name                | TEXT     |                                      | false    |                                           |
| side                | TEXT     |                                      | false    |                                           |
| active              | INTEGER  | 0                                    | false    |                                           |
| defeated            | INTEGER  | 0                                    | false    |                                           |
| position            | INTEGER  |                                      | false    |                                           |
| created_at          | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                           |
| updated_at          | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                           |
| deleted_at          | DATETIME |                                      | true     |                                           |

## Constraints

| Name                          | Type        | Definition                                                                                                             |
| ----------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------- |
| id                            | PRIMARY KEY | PRIMARY KEY (id)                                                                                                       |
| - (Foreign key ID: 0)         | FOREIGN KEY | FOREIGN KEY (player_character_id) REFERENCES player_characters (id) ON UPDATE NO ACTION ON DELETE NO ACTION MATCH NONE |
| - (Foreign key ID: 1)         | FOREIGN KEY | FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles (id) ON UPDATE NO ACTION ON DELETE NO ACTION MATCH NONE         |
| - (Foreign key ID: 2)         | FOREIGN KEY | FOREIGN KEY (encounter_id) REFERENCES encounters (id) ON UPDATE NO ACTION ON DELETE CASCADE MATCH NONE                 |
| sqlite_autoindex_combatants_1 | PRIMARY KEY | PRIMARY KEY (id)                                                                                                       |
| -                             | CHECK       | CHECK (trim(id) <> '')                                                                                                 |
| -                             | CHECK       | CHECK (trim(encounter_id) <> '')                                                                                       |
| -                             | CHECK       | CHECK (trim(stat_profile_id) <> '')                                                                                    |
| -                             | CHECK       | CHECK (trim(name) <> '')                                                                                               |
| -                             | CHECK       | CHECK (side IN ('party', 'npc'))                                                                                       |
| -                             | CHECK       | CHECK (active IN (0, 1))                                                                                               |
| -                             | CHECK       | CHECK (defeated IN (0, 1))                                                                                             |
| -                             | CHECK       | CHECK (position >= 0)                                                                                                  |

## Indexes

| Name                                      | Definition                                                                                                                                                     |
| ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| idx_combatants_encounter_player_character | CREATE UNIQUE INDEX idx_combatants_encounter_player_character<br />ON combatants(encounter_id, player_character_id)<br />WHERE player_character_id IS NOT NULL |
| idx_combatants_encounter_position         | CREATE INDEX idx_combatants_encounter_position<br />ON combatants(encounter_id, position)                                                                      |
| sqlite_autoindex_combatants_1             | PRIMARY KEY (id)                                                                                                                                               |

## Triggers

| Name                               | Definition                                                                                                                                                                        |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_combatants_delete_stat_profile | CREATE TRIGGER trg_combatants_delete_stat_profile<br />AFTER DELETE ON combatants<br />BEGIN<br />    DELETE FROM stat_profiles<br />    WHERE id = OLD.stat_profile_id;<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
