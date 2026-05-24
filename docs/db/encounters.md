# encounters

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE "encounters" (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NULL CHECK (campaign_id IS NULL OR trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    turn_index INTEGER NOT NULL CHECK (turn_index >= 0),
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown' CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly')),
    difficulty_score REAL NOT NULL DEFAULT 0 CHECK (difficulty_score >= 0),
    party_count INTEGER NOT NULL DEFAULT 0 CHECK (party_count >= 0),
    party_avg_level REAL NOT NULL DEFAULT 0 CHECK (party_avg_level >= 0),
    party_xp_budget INTEGER NOT NULL DEFAULT 0 CHECK (party_xp_budget >= 0),
    enemy_count INTEGER NOT NULL DEFAULT 0 CHECK (enemy_count >= 0),
    enemy_avg_level REAL NOT NULL DEFAULT 0 CHECK (enemy_avg_level >= 0),
    enemy_total_xp INTEGER NOT NULL DEFAULT 0 CHECK (enemy_total_xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
)
```

</details>

## Columns

| Name             | Type     | Default                              | Nullable | Children                                                        |
| ---------------- | -------- | ------------------------------------ | -------- | --------------------------------------------------------------- |
| id               | TEXT     |                                      | true     | [encounter_logs](encounter_logs.md) [combatants](combatants.md) |
| campaign_id      | TEXT     |                                      | true     |                                                                 |
| name             | TEXT     |                                      | false    |                                                                 |
| round            | INTEGER  |                                      | false    |                                                                 |
| turn_index       | INTEGER  |                                      | false    |                                                                 |
| party_ap         | INTEGER  | 0                                    | false    |                                                                 |
| gm_threat        | INTEGER  | 0                                    | false    |                                                                 |
| difficulty_label | TEXT     | 'Unknown'                            | false    |                                                                 |
| difficulty_score | REAL     | 0                                    | false    |                                                                 |
| party_count      | INTEGER  | 0                                    | false    |                                                                 |
| party_avg_level  | REAL     | 0                                    | false    |                                                                 |
| party_xp_budget  | INTEGER  | 0                                    | false    |                                                                 |
| enemy_count      | INTEGER  | 0                                    | false    |                                                                 |
| enemy_avg_level  | REAL     | 0                                    | false    |                                                                 |
| enemy_total_xp   | INTEGER  | 0                                    | false    |                                                                 |
| created_at       | DATETIME | STRFTIME('%Y-%m-%d %H:%M:%f', 'now') | false    |                                                                 |
| updated_at       | DATETIME | CURRENT_TIMESTAMP                    | false    |                                                                 |
| deleted_at       | DATETIME |                                      | true     |                                                                 |

## Constraints

| Name                          | Type        | Definition                                                                             |
| ----------------------------- | ----------- | -------------------------------------------------------------------------------------- |
| id                            | PRIMARY KEY | PRIMARY KEY (id)                                                                       |
| sqlite_autoindex_encounters_1 | PRIMARY KEY | PRIMARY KEY (id)                                                                       |
| -                             | CHECK       | CHECK (trim(id) <> '')                                                                 |
| -                             | CHECK       | CHECK (campaign_id IS NULL OR trim(campaign_id) <> '')                                 |
| -                             | CHECK       | CHECK (trim(name) <> '')                                                               |
| -                             | CHECK       | CHECK (round >= 1)                                                                     |
| -                             | CHECK       | CHECK (turn_index >= 0)                                                                |
| -                             | CHECK       | CHECK (party_ap >= 0)                                                                  |
| -                             | CHECK       | CHECK (gm_threat >= 0)                                                                 |
| -                             | CHECK       | CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly')) |
| -                             | CHECK       | CHECK (difficulty_score >= 0)                                                          |
| -                             | CHECK       | CHECK (party_count >= 0)                                                               |
| -                             | CHECK       | CHECK (party_avg_level >= 0)                                                           |
| -                             | CHECK       | CHECK (party_xp_budget >= 0)                                                           |
| -                             | CHECK       | CHECK (enemy_count >= 0)                                                               |
| -                             | CHECK       | CHECK (enemy_avg_level >= 0)                                                           |
| -                             | CHECK       | CHECK (enemy_total_xp >= 0)                                                            |

## Indexes

| Name                                    | Definition                                                                                                                 |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| idx_encounters_campaign_deleted_updated | CREATE INDEX idx_encounters_campaign_deleted_updated<br />ON encounters(campaign_id, deleted_at, updated_at DESC, id DESC) |
| sqlite_autoindex_encounters_1           | PRIMARY KEY (id)                                                                                                           |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
