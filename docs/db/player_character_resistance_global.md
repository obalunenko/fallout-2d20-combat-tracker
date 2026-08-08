# player_character_resistance_global

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW player_character_resistance_global AS
SELECT
    pc.id AS player_character_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = pc.stat_profile_id
WHERE spr.body_location_id = 0
```

</details>

## Columns

| Name                | Type     | Default | Nullable |
| ------------------- | -------- | ------- | -------- |
| player_character_id | TEXT     |         | true     |
| damage_type_id      | INTEGER  |         | true     |
| resistance          | INTEGER  |         | true     |
| immune              | INTEGER  |         | true     |
| created_at          | DATETIME |         | true     |
| updated_at          | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [player_characters](player_characters.md)                                     | 10      |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 7       |         | table |

## Triggers

| Name                                                 | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_compat_player_character_resistance_global_insert | CREATE TRIGGER trg_compat_player_character_resistance_global_insert<br />INSTEAD OF INSERT ON player_character_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown player_character_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        pc.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM player_characters pc<br />    WHERE pc.id = NEW.player_character_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                                     |
| trg_compat_player_character_resistance_global_update | CREATE TRIGGER trg_compat_player_character_resistance_global_update<br />INSTEAD OF UPDATE ON player_character_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown player_character_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        pc.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM player_characters pc<br />    WHERE pc.id = NEW.player_character_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_player_character_resistance_global_delete | CREATE TRIGGER trg_compat_player_character_resistance_global_delete<br />INSTEAD OF DELETE ON player_character_resistance_global<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
