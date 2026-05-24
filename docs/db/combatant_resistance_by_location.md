# combatant_resistance_by_location

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW combatant_resistance_by_location AS
SELECT
    c.id AS combatant_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM combatants c
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = c.stat_profile_id
```

</details>

## Columns

| Name             | Type     | Default | Nullable |
| ---------------- | -------- | ------- | -------- |
| combatant_id     | TEXT     |         | true     |
| damage_type_id   | INTEGER  |         | true     |
| body_location_id | INTEGER  |         | true     |
| resistance       | INTEGER  |         | true     |
| created_at       | DATETIME |         | true     |
| updated_at       | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [combatants](combatants.md)                                                   | 12      |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 6       |         | table |

## Triggers

| Name                                               | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| trg_compat_combatant_resistance_by_location_insert | CREATE TRIGGER trg_compat_combatant_resistance_by_location_insert<br />INSTEAD OF INSERT ON combatant_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown combatant_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        c.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM combatants c<br />    WHERE c.id = NEW.combatant_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                                       |
| trg_compat_combatant_resistance_by_location_update | CREATE TRIGGER trg_compat_combatant_resistance_by_location_update<br />INSTEAD OF UPDATE ON combatant_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown combatant_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        c.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM combatants c<br />    WHERE c.id = NEW.combatant_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_combatant_resistance_by_location_delete | CREATE TRIGGER trg_compat_combatant_resistance_by_location_delete<br />INSTEAD OF DELETE ON combatant_resistance_by_location<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
