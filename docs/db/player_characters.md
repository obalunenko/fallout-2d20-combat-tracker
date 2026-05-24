# player_characters

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "player_characters" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    player_id TEXT NOT NULL CHECK (trim(player_id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
)
```

</details>

## Columns

| Name                | Type     | Default                              | Nullable | Children                    | Parents                           |
| ------------------- | -------- | ------------------------------------ | -------- | --------------------------- | --------------------------------- |
| id                  | TEXT     |                                      | true     | [combatants](combatants.md) |                                   |
| player_id           | TEXT     |                                      | false    |                             | [players](players.md)             |
| campaign_id         | TEXT     |                                      | false    |                             | [campaigns](campaigns.md)         |
| stat_profile_id     | TEXT     |                                      | false    |                             | [stat_profiles](stat_profiles.md) |
| name                | TEXT     |                                      | false    |                             |                                   |
| active              | INTEGER  | 1                                    | false    |                             |                                   |
| availability_status | TEXT     | 'active'                             | false    |                             |                                   |
| created_at          | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                             |                                   |
| updated_at          | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                             |                                   |
| deleted_at          | DATETIME |                                      | true     |                             |                                   |

## Constraints

| Name                                 | Type        | Definition                                                                                                     |
| ------------------------------------ | ----------- | -------------------------------------------------------------------------------------------------------------- |
| id                                   | PRIMARY KEY | PRIMARY KEY (id)                                                                                               |
| - (Foreign key ID: 0)                | FOREIGN KEY | FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles (id) ON UPDATE NO ACTION ON DELETE NO ACTION MATCH NONE |
| - (Foreign key ID: 1)                | FOREIGN KEY | FOREIGN KEY (campaign_id) REFERENCES campaigns (id) ON UPDATE NO ACTION ON DELETE CASCADE MATCH NONE           |
| - (Foreign key ID: 2)                | FOREIGN KEY | FOREIGN KEY (player_id) REFERENCES players (id) ON UPDATE NO ACTION ON DELETE CASCADE MATCH NONE               |
| sqlite_autoindex_player_characters_1 | PRIMARY KEY | PRIMARY KEY (id)                                                                                               |
| -                                    | CHECK       | CHECK (trim(id) <> '')                                                                                         |
| -                                    | CHECK       | CHECK (trim(player_id) <> '')                                                                                  |
| -                                    | CHECK       | CHECK (trim(campaign_id) <> '')                                                                                |
| -                                    | CHECK       | CHECK (trim(stat_profile_id) <> '')                                                                            |
| -                                    | CHECK       | CHECK (trim(name) <> '')                                                                                       |
| -                                    | CHECK       | CHECK (active IN (0, 1))                                                                                       |
| -                                    | CHECK       | CHECK (availability_status IN ('active', 'inactive'))                                                          |

## Indexes

| Name                                        | Definition                                                                                                                         |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| idx_player_characters_campaign_availability | CREATE INDEX idx_player_characters_campaign_availability<br />ON player_characters(campaign_id, active, availability_status, name) |
| idx_player_characters_campaign_active       | CREATE INDEX idx_player_characters_campaign_active<br />ON player_characters(campaign_id, active, name)                            |
| idx_player_characters_one_active            | CREATE UNIQUE INDEX idx_player_characters_one_active<br />ON player_characters(player_id)<br />WHERE active = 1                    |
| sqlite_autoindex_player_characters_1        | PRIMARY KEY (id)                                                                                                                   |

## Triggers

| Name                                      | Definition                                                                                                                                                                                                                                                               |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_player_characters_delete_stat_profile | CREATE TRIGGER trg_player_characters_delete_stat_profile<br />AFTER DELETE ON player_characters<br />BEGIN<br />    DELETE FROM stat_profiles<br />    WHERE id = OLD.stat_profile_id;<br />END                                                                          |
| trg_player_characters_require_level       | CREATE TRIGGER trg_player_characters_require_level<br />BEFORE INSERT ON player_characters<br />WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1<br />BEGIN<br />    SELECT RAISE(ABORT, 'player character level must be at least 1');<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
