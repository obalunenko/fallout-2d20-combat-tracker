CREATE TABLE IF NOT EXISTS encounters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    round INTEGER NOT NULL,
    turn_index INTEGER NOT NULL,
    party_ap INTEGER NOT NULL DEFAULT 0,
    gm_threat INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS combatants (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    name TEXT NOT NULL,
    side TEXT NOT NULL,
    initiative INTEGER NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    defeated INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL,
    hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    damage_resistance INTEGER NOT NULL DEFAULT 0,
    damage_resistance_physical INTEGER NOT NULL DEFAULT 0,
    damage_resistance_energy INTEGER NOT NULL DEFAULT 0,
    damage_resistance_radiation INTEGER NOT NULL DEFAULT 0,
    damage_resistance_poison INTEGER NOT NULL DEFAULT 0,
    damage_resistance_physical_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_energy_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_radiation_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_poison_immune INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    xp INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);

CREATE INDEX IF NOT EXISTS idx_encounters_deleted_updated
ON encounters(deleted_at, updated_at DESC);

CREATE TABLE IF NOT EXISTS encounter_logs (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    round INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);
