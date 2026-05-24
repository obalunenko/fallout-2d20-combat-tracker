# Database Normalization Notes

This document separates canonical relational state from deliberate cached or
snapshot state. The generated schema and table docs describe what exists; this
file describes which data is intended to be a source of truth.

## Canonical State

These tables and columns are the primary relational state:

| Area | Canonical storage |
| ---- | ----------------- |
| Campaigns | `campaigns` |
| Players | `players` |
| Player characters | `player_characters` plus their `stat_profiles` row |
| Encounters | `encounters` for encounter identity, round, turn index, AP, and threat |
| Combatants | `combatants` for encounter membership/order; their `stat_profiles` row is canonical for NPC and unlinked combatants |
| Monster templates | `monster_templates` plus their `stat_profiles` row |
| Damage and location dictionaries | `damage_types`, `body_locations` |
| Resistance values | `stat_profile_resistance_global`, `stat_profile_resistance_by_location` |

`stat_profiles` is the canonical place for level, XP, initiative, HP, max HP,
defense, and torso-only state. Owner tables reference stat profiles instead of
duplicating those columns.

Resistance data is normalized by stat profile, damage type, and optionally body
location. The `combatant_*`, `player_character_*`, and `monster_template_*`
resistance views are compatibility views over the normalized stat-profile
tables.

`player_characters` does not store `campaign_id`; a character's campaign is
derived through `player_characters.player_id -> players.campaign_id`.

## Deliberate Caches And Snapshots

The following fields are intentionally stored even though they can be derived
from other tables:

| Field(s) | Type | Reason | Refresh path |
| -------- | ---- | ------ | ------------ |
| `encounters.difficulty_label`, `difficulty_score`, `party_count`, `party_avg_level`, `party_xp_budget`, `enemy_count`, `enemy_avg_level`, `enemy_total_xp` | Summary cache | Encounter lists need difficulty metrics without rebuilding every combatant profile for each row. | Recomputed by `saveEncounter` through `domain.EvaluateEncounterDifficulty`. |
| `combatants.position` | Ordered snapshot | Preserves encounter initiative order without depending on mutable stats or names. | Rewritten whenever an encounter is saved. |
| `combatants.active` | Turn marker cache | Mirrors the active turn for fast row-level reads. The stricter source-of-truth decision belongs to the active-turn normalization step. | Rewritten whenever an encounter is saved. |
| `combatants.name` for linked party combatants | Snapshot/fallback | Keeps a combatant-readable name if the party character link is removed or the row is inspected directly. Normal app reads prefer the linked player character name. | Rewritten whenever an encounter is saved. |
| `combatants.stat_profile_id` for linked party combatants | Snapshot/fallback | Every combatant owns a profile for uniform lifecycle and fallback reads. Normal app reads prefer the linked player character profile when `player_character_id` is present. | Rewritten whenever an encounter is saved. |
| `monster_templates.name_key` | Derived lookup key | Provides stable case-insensitive upsert behavior for template names. | Written by repository mappers from normalized names. |

## Consistency Rules

Repository writes are the synchronization boundary. Code that mutates canonical
rows directly must also refresh dependent cache rows, or should use the
repository APIs instead.

The current schema enforces many relationship invariants with foreign keys,
unique indexes, and triggers, but not every cached value can be validated by
SQLite constraints. In particular, `encounters` difficulty metrics are trusted
to be refreshed by the application on save.

If stricter normalization becomes more important than summary-read speed, the
encounter difficulty cache should be replaced with a view or query-level
calculation.
