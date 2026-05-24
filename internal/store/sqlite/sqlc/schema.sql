CREATE TABLE IF NOT EXISTS encounters (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NULL CHECK (campaign_id IS NULL OR trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    turn_index INTEGER NOT NULL CHECK (turn_index >= 0),
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown' CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly')),
    difficulty_score REAL NOT NULL DEFAULT 0 CHECK (difficulty_score >= 0),
    party_count INTEGER NOT NULL DEFAULT 0 CHECK (party_count >= 0),
    party_avg_level REAL NOT NULL DEFAULT 0 CHECK (party_avg_level >= 0),
    party_xp_budget INTEGER NOT NULL DEFAULT 0 CHECK (party_xp_budget >= 0),
    enemy_count INTEGER NOT NULL DEFAULT 0 CHECK (enemy_count >= 0),
    enemy_avg_level REAL NOT NULL DEFAULT 0 CHECK (enemy_avg_level >= 0),
    enemy_total_xp INTEGER NOT NULL DEFAULT 0 CHECK (enemy_total_xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS combatants (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    player_character_id TEXT NULL,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    side TEXT NOT NULL CHECK (side IN ('party', 'npc')),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    initiative INTEGER NOT NULL CHECK (initiative >= 0),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
    defeated INTEGER NOT NULL DEFAULT 0 CHECK (defeated IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 0),
    xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE,
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id)
);

CREATE TABLE IF NOT EXISTS body_locations (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
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

CREATE TRIGGER IF NOT EXISTS trg_combatant_resistance_global_poison_only_insert
BEFORE INSERT ON combatant_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

CREATE TRIGGER IF NOT EXISTS trg_combatant_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON combatant_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

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

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_encounter_player_character
ON combatants(encounter_id, player_character_id)
WHERE player_character_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_encounters_deleted_updated
ON encounters(deleted_at, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC);

CREATE TABLE IF NOT EXISTS encounter_logs (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    message TEXT NOT NULL CHECK (trim(message) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS campaigns (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
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
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_players_campaign_name
ON players(campaign_id, lower(trim(name)));

CREATE TABLE IF NOT EXISTS player_characters (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    player_id TEXT NOT NULL CHECK (trim(player_id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 1),
    initiative INTEGER NOT NULL DEFAULT 1 CHECK (initiative >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp),
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
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

CREATE TRIGGER IF NOT EXISTS trg_player_character_resistance_global_poison_only_insert
BEFORE INSERT ON player_character_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

CREATE TRIGGER IF NOT EXISTS trg_player_character_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON player_character_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

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

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

CREATE TABLE IF NOT EXISTS monster_templates (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    name_key TEXT NOT NULL UNIQUE CHECK (trim(name_key) <> ''),
    torso_only INTEGER NOT NULL DEFAULT 1 CHECK (torso_only IN (0, 1)),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 1),
    xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
    initiative INTEGER NOT NULL DEFAULT 1 CHECK (initiative >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp)
);

CREATE INDEX IF NOT EXISTS idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS monster_template_resistance_global (
    monster_template_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (monster_template_id, damage_type_id),
    FOREIGN KEY (monster_template_id) REFERENCES monster_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

CREATE TRIGGER IF NOT EXISTS trg_monster_template_resistance_global_poison_only_insert
BEFORE INSERT ON monster_template_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

CREATE TRIGGER IF NOT EXISTS trg_monster_template_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON monster_template_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;

CREATE TABLE IF NOT EXISTS monster_template_resistance_by_location (
    monster_template_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (monster_template_id, damage_type_id, body_location_id),
    FOREIGN KEY (monster_template_id) REFERENCES monster_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);
