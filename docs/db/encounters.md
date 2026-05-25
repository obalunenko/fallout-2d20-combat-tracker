# encounters

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "encounters" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    turn_index INTEGER NOT NULL CHECK (turn_index >= 0),
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
)
```

</details>

## Columns

| Name        | Type     | Default                              | Nullable | Children                                                        | Parents                   |
| ----------- | -------- | ------------------------------------ | -------- | --------------------------------------------------------------- | ------------------------- |
| id          | TEXT     |                                      | true     | [encounter_logs](encounter_logs.md) [combatants](combatants.md) |                           |
| campaign_id | TEXT     |                                      | false    |                                                                 | [campaigns](campaigns.md) |
| name        | TEXT     |                                      | false    |                                                                 |                           |
| round       | INTEGER  |                                      | false    |                                                                 |                           |
| turn_index  | INTEGER  |                                      | false    |                                                                 |                           |
| party_ap    | INTEGER  | 0                                    | false    |                                                                 |                           |
| gm_threat   | INTEGER  | 0                                    | false    |                                                                 |                           |
| created_at  | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                                                 |                           |
| updated_at  | DATETIME | CURRENT_TIMESTAMP                    | false    |                                                                 |                           |
| deleted_at  | DATETIME |                                      | true     |                                                                 |                           |

## Constraints

| Name                          | Type        | Definition                                                                                           |
| ----------------------------- | ----------- | ---------------------------------------------------------------------------------------------------- |
| id                            | PRIMARY KEY | PRIMARY KEY (id)                                                                                     |
| - (Foreign key ID: 0)         | FOREIGN KEY | FOREIGN KEY (campaign_id) REFERENCES campaigns (id) ON UPDATE NO ACTION ON DELETE CASCADE MATCH NONE |
| sqlite_autoindex_encounters_1 | PRIMARY KEY | PRIMARY KEY (id)                                                                                     |
| -                             | CHECK       | CHECK (trim(id) <> '')                                                                               |
| -                             | CHECK       | CHECK (trim(campaign_id) <> '')                                                                      |
| -                             | CHECK       | CHECK (trim(name) <> '')                                                                             |
| -                             | CHECK       | CHECK (round >= 1)                                                                                   |
| -                             | CHECK       | CHECK (turn_index >= 0)                                                                              |
| -                             | CHECK       | CHECK (party_ap >= 0)                                                                                |
| -                             | CHECK       | CHECK (gm_threat >= 0)                                                                               |

## Indexes

| Name                                    | Definition                                                                                                                 |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| idx_encounters_campaign_deleted_updated | CREATE INDEX idx_encounters_campaign_deleted_updated<br />ON encounters(campaign_id, deleted_at, updated_at DESC, id DESC) |
| sqlite_autoindex_encounters_1           | PRIMARY KEY (id)                                                                                                           |

## Triggers

| Name                                                       | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_encounters_campaign_update_keeps_combatants_consistent | CREATE TRIGGER trg_encounters_campaign_update_keeps_combatants_consistent<br />BEFORE UPDATE OF campaign_id ON encounters<br />WHEN EXISTS (<br />    SELECT 1<br />    FROM combatants c<br />    JOIN player_characters pc ON pc.id = c.player_character_id<br />    JOIN players p ON p.id = pc.player_id<br />    WHERE c.encounter_id = NEW.id<br />      AND p.campaign_id <> NEW.campaign_id<br />)<br />BEGIN<br />    SELECT RAISE(ABORT, 'encounter campaign update would mismatch linked combatants');<br />END |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
