# monster_template_resistance_global

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW monster_template_resistance_global AS
SELECT
    mt.id AS monster_template_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = mt.stat_profile_id
WHERE spr.body_location_id = 0
```

</details>

## Columns

| Name                | Type     | Default | Nullable |
| ------------------- | -------- | ------- | -------- |
| monster_template_id | TEXT     |         | true     |
| damage_type_id      | INTEGER  |         | true     |
| resistance          | INTEGER  |         | true     |
| immune              | INTEGER  |         | true     |
| created_at          | DATETIME |         | true     |
| updated_at          | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [monster_templates](monster_templates.md)                                     | 7       |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 7       |         | table |

## Triggers

| Name                                                 | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_compat_monster_template_resistance_global_insert | CREATE TRIGGER trg_compat_monster_template_resistance_global_insert<br />INSTEAD OF INSERT ON monster_template_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown monster_template_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        mt.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM monster_templates mt<br />    WHERE mt.id = NEW.monster_template_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                                     |
| trg_compat_monster_template_resistance_global_update | CREATE TRIGGER trg_compat_monster_template_resistance_global_update<br />INSTEAD OF UPDATE ON monster_template_resistance_global<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown monster_template_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at<br />    )<br />    SELECT<br />        mt.stat_profile_id,<br />        NEW.damage_type_id,<br />        0,<br />        NEW.resistance,<br />        NEW.immune,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM monster_templates mt<br />    WHERE mt.id = NEW.monster_template_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        immune = excluded.immune,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_monster_template_resistance_global_delete | CREATE TRIGGER trg_compat_monster_template_resistance_global_delete<br />INSTEAD OF DELETE ON monster_template_resistance_global<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = 0;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
