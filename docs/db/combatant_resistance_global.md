# combatant_resistance_global

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW combatant_resistance_global AS
SELECT
    c.id AS combatant_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM combatants c
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = c.stat_profile_id
WHERE spr.body_location_id = 0
```

</details>

## Columns

| Name           | Type     | Default | Nullable |
| -------------- | -------- | ------- | -------- |
| combatant_id   | TEXT     |         | true     |
| damage_type_id | INTEGER  |         | true     |
| resistance     | INTEGER  |         | true     |
| immune         | INTEGER  |         | true     |
| created_at     | DATETIME |         | true     |
| updated_at     | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [combatants](combatants.md)                                                   | 11      |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 7       |         | table |

## Triggers

| Name                                          | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_compat_combatant_resistance_global_insert | CREATE TRIGGER trg_compat_combatant_resistance_global_insert<br />INSTEAD OF INSERT ON combatant_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown combatant_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        c.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM combatants c<br />    WHERE c.id = NEW.combatant_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                    |
| trg_compat_combatant_resistance_global_update | CREATE TRIGGER trg_compat_combatant_resistance_global_update<br />INSTEAD OF UPDATE ON combatant_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown combatant_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        c.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM combatants c<br />    WHERE c.id = NEW.combatant_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_combatant_resistance_global_delete | CREATE TRIGGER trg_compat_combatant_resistance_global_delete<br />INSTEAD OF DELETE ON combatant_resistance_global<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
