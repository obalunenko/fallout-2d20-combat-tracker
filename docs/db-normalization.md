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
| Player characters | `player_characters` (identity, availability, exact notes), their `stat_profiles` row, and `player_character_special_attributes` |
| Encounters | `encounters` for encounter identity, round, active turn index, AP, and threat |
| Combatants | `combatants` for encounter membership/order; their `stat_profiles` row is canonical for NPC and unlinked combatants |
| Monster templates | `monster_templates` plus their `stat_profiles` row |
| Damage and location dictionaries | `damage_types`, `body_locations` |
| Resistance values | `stat_profile_resistance_by_location` |

`stat_profiles` is the canonical place for level, XP, initiative, HP, max HP,
defense, and torso-only state. Owner tables reference stat profiles instead of
duplicating those columns.

Resistance data is normalized by stat profile, damage type, and body location.
Global resistance uses the `body_locations.code = 'global'` dictionary row in
`stat_profile_resistance_by_location`. For the same stat profile and damage
type, a `global` row is mutually exclusive with any concrete body-location row.
The `combatant_*`, `player_character_*`, and `monster_template_*` resistance
views are compatibility views over the normalized stat-profile table.

`player_characters` does not store `campaign_id`; a character's campaign is
derived through `player_characters.player_id -> players.campaign_id`.

S.P.E.C.I.A.L. is normalized through the stable seven-row
`special_attributes` dictionary and one positive integer row per character and
attribute in `player_character_special_attributes`. Migration `00043` gives
existing characters blank notes and a complete profile with every value set to
1. Notes remain owned by `player_characters` and are stored without trimming so
intentional whitespace and line breaks survive a round trip.

`combatants` does not store `active`; active combatants are derived from
`combatants.position = encounters.turn_index`.

## Deliberate Caches And Snapshots

The following fields are intentionally stored even though they can be derived
from other tables:

| Field(s) | Type | Reason | Refresh path |
| -------- | ---- | ------ | ------------ |
| `combatants.position` | Ordered snapshot | Preserves encounter initiative order without depending on mutable stats or names. | Rewritten whenever an encounter is saved. |
| `combatants.name` for linked party combatants | Snapshot | Preserves the name recorded for that encounter. | Rewritten whenever an encounter is explicitly saved. |
| `combatants.stat_profile_id` for linked party combatants | Combat snapshot | Stores the encounter's scalar and resistance state. Campaign edits copy level, HP, max HP, Defense, DR/immunity, and defeated state only into the effective active encounter; closed encounters remain historical. | Rewritten on explicit encounter save; selectively synchronized for the active campaign's latest activated encounter. |

## Consistency Rules

Repository writes are the synchronization boundary. Code that mutates canonical
rows directly must also refresh dependent cache rows, or should use the
repository APIs instead.

The current schema enforces many relationship invariants with foreign keys,
unique indexes, and triggers. Encounter difficulty metrics are calculated at
read time from combatant profiles with `domain.EvaluateEncounterDifficulty`
rather than stored in `encounters`.
