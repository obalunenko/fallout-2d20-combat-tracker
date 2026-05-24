# player_character_resistance_by_location

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW player_character_resistance_by_location AS
SELECT
    pc.id AS player_character_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = pc.stat_profile_id
```

</details>

## Columns

| Name                | Type     | Default | Nullable |
| ------------------- | -------- | ------- | -------- |
| player_character_id | TEXT     |         | true     |
| damage_type_id      | INTEGER  |         | true     |
| body_location_id    | INTEGER  |         | true     |
| resistance          | INTEGER  |         | true     |
| created_at          | DATETIME |         | true     |
| updated_at          | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [player_characters](player_characters.md)                                     | 9       |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 6       |         | table |

## Triggers

| Name                                                      | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_compat_player_character_resistance_by_location_insert | CREATE TRIGGER trg_compat_player_character_resistance_by_location_insert<br />INSTEAD OF INSERT ON player_character_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown player_character_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        pc.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM player_characters pc<br />    WHERE pc.id = NEW.player_character_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                                                        |
| trg_compat_player_character_resistance_by_location_update | CREATE TRIGGER trg_compat_player_character_resistance_by_location_update<br />INSTEAD OF UPDATE ON player_character_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown player_character_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        pc.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM player_characters pc<br />    WHERE pc.id = NEW.player_character_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_player_character_resistance_by_location_delete | CREATE TRIGGER trg_compat_player_character_resistance_by_location_delete<br />INSTEAD OF DELETE ON player_character_resistance_by_location<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
