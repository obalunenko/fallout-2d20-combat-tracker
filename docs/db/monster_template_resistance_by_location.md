# monster_template_resistance_by_location

## Description

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE VIEW monster_template_resistance_by_location AS
SELECT
    mt.id AS monster_template_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = mt.stat_profile_id
```

</details>

## Columns

| Name                | Type     | Default | Nullable |
| ------------------- | -------- | ------- | -------- |
| monster_template_id | TEXT     |         | true     |
| damage_type_id      | INTEGER  |         | true     |
| body_location_id    | INTEGER  |         | true     |
| resistance          | INTEGER  |         | true     |
| created_at          | DATETIME |         | true     |
| updated_at          | DATETIME |         | true     |

## Referenced Tables

| Name                                                                          | Columns | Comment | Type  |
| ----------------------------------------------------------------------------- | ------- | ------- | ----- |
| [monster_templates](monster_templates.md)                                     | 7       |         | table |
| [stat_profile_resistance_by_location](stat_profile_resistance_by_location.md) | 6       |         | table |

## Triggers

| Name                                                      | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| trg_compat_monster_template_resistance_by_location_insert | CREATE TRIGGER trg_compat_monster_template_resistance_by_location_insert<br />INSTEAD OF INSERT ON monster_template_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown monster_template_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        mt.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM monster_templates mt<br />    WHERE mt.id = NEW.monster_template_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END                                                                                                                                                                                                                                                                                                                        |
| trg_compat_monster_template_resistance_by_location_update | CREATE TRIGGER trg_compat_monster_template_resistance_by_location_update<br />INSTEAD OF UPDATE ON monster_template_resistance_by_location<br />BEGIN<br />    SELECT RAISE(ABORT, 'unknown monster_template_id')<br />    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);<br /><br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br /><br />    INSERT INTO stat_profile_resistance_by_location (<br />        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at<br />    )<br />    SELECT<br />        mt.stat_profile_id,<br />        NEW.damage_type_id,<br />        NEW.body_location_id,<br />        NEW.resistance,<br />        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),<br />        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))<br />    FROM monster_templates mt<br />    WHERE mt.id = NEW.monster_template_id<br />    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET<br />        resistance = excluded.resistance,<br />        updated_at = excluded.updated_at;<br />END |
| trg_compat_monster_template_resistance_by_location_delete | CREATE TRIGGER trg_compat_monster_template_resistance_by_location_delete<br />INSTEAD OF DELETE ON monster_template_resistance_by_location<br />BEGIN<br />    DELETE FROM stat_profile_resistance_by_location<br />    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)<br />      AND damage_type_id = OLD.damage_type_id<br />      AND body_location_id = OLD.body_location_id;<br />END                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
