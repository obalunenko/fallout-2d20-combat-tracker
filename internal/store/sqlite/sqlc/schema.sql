CREATE TABLE IF NOT EXISTS encounters (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NULL,
    name TEXT NOT NULL,
    round INTEGER NOT NULL,
    turn_index INTEGER NOT NULL,
    party_ap INTEGER NOT NULL DEFAULT 0,
    gm_threat INTEGER NOT NULL DEFAULT 0,
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown',
    difficulty_score REAL NOT NULL DEFAULT 0,
    party_count INTEGER NOT NULL DEFAULT 0,
    party_avg_level REAL NOT NULL DEFAULT 0,
    party_xp_budget INTEGER NOT NULL DEFAULT 0,
    enemy_count INTEGER NOT NULL DEFAULT 0,
    enemy_avg_level REAL NOT NULL DEFAULT 0,
    enemy_total_xp INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS combatants (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    name TEXT NOT NULL,
    side TEXT NOT NULL,
    torso_only INTEGER NOT NULL DEFAULT 0,
    initiative INTEGER NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    defeated INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL,
    hp INTEGER NOT NULL DEFAULT 1,
    max_hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    xp INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS body_locations (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS combatant_defense_by_location (
    combatant_id TEXT NOT NULL,
    body_location_id INTEGER NOT NULL,
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, body_location_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE TABLE IF NOT EXISTS damage_types (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS combatant_resistance_global (
    combatant_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, damage_type_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

CREATE TABLE IF NOT EXISTS combatant_resistance_by_location (
    combatant_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, damage_type_id, body_location_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);

CREATE INDEX IF NOT EXISTS idx_encounters_deleted_updated
ON encounters(deleted_at, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC);

CREATE TABLE IF NOT EXISTS encounter_logs (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    round INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    start_date DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS app_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    active_campaign_id TEXT NULL,
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (active_campaign_id) REFERENCES campaigns(id)
);

CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_players_campaign_name
ON players(campaign_id, lower(trim(name)));

CREATE TABLE IF NOT EXISTS player_characters (
    id TEXT PRIMARY KEY,
    player_id TEXT NOT NULL,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    initiative INTEGER NOT NULL DEFAULT 1,
    hp INTEGER NOT NULL DEFAULT 1,
    max_hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    torso_only INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS player_character_defense_by_location (
    player_character_id TEXT NOT NULL,
    body_location_id INTEGER NOT NULL,
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, body_location_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE TABLE IF NOT EXISTS player_character_resistance_global (
    player_character_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, damage_type_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

CREATE TABLE IF NOT EXISTS player_character_resistance_by_location (
    player_character_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, damage_type_id, body_location_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);
