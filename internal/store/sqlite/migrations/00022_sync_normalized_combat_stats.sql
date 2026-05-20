-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS trg_sync_combatant_stats_after_insert
AFTER INSERT ON combatants
BEGIN
    DELETE FROM combatant_defense_by_location
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_defense_by_location (combatant_id, body_location_id, defense, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.defense_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.defense_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.defense_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.defense_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 5, NEW.defense_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 6, NEW.defense_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM combatant_resistance_global
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.damage_resistance_physical, NEW.damage_resistance_physical_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.damage_resistance_energy, NEW.damage_resistance_energy_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.damage_resistance_radiation, NEW.damage_resistance_radiation_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.damage_resistance_poison, NEW.damage_resistance_poison_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM combatant_resistance_by_location
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_resistance_by_location (combatant_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
    VALUES
        (NEW.id, 1, 1, NEW.damage_resistance_physical_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 2, NEW.damage_resistance_physical_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 3, NEW.damage_resistance_physical_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 4, NEW.damage_resistance_physical_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 5, NEW.damage_resistance_physical_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 6, NEW.damage_resistance_physical_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 1, NEW.damage_resistance_energy_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 2, NEW.damage_resistance_energy_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 3, NEW.damage_resistance_energy_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 4, NEW.damage_resistance_energy_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 5, NEW.damage_resistance_energy_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 6, NEW.damage_resistance_energy_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 1, NEW.damage_resistance_radiation_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 2, NEW.damage_resistance_radiation_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 3, NEW.damage_resistance_radiation_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 4, NEW.damage_resistance_radiation_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 5, NEW.damage_resistance_radiation_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 6, NEW.damage_resistance_radiation_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS trg_sync_combatant_stats_after_update
AFTER UPDATE ON combatants
BEGIN
    DELETE FROM combatant_defense_by_location
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_defense_by_location (combatant_id, body_location_id, defense, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.defense_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.defense_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.defense_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.defense_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 5, NEW.defense_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 6, NEW.defense_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM combatant_resistance_global
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.damage_resistance_physical, NEW.damage_resistance_physical_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.damage_resistance_energy, NEW.damage_resistance_energy_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.damage_resistance_radiation, NEW.damage_resistance_radiation_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.damage_resistance_poison, NEW.damage_resistance_poison_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM combatant_resistance_by_location
    WHERE combatant_id = NEW.id;

    INSERT INTO combatant_resistance_by_location (combatant_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
    VALUES
        (NEW.id, 1, 1, NEW.damage_resistance_physical_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 2, NEW.damage_resistance_physical_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 3, NEW.damage_resistance_physical_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 4, NEW.damage_resistance_physical_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 5, NEW.damage_resistance_physical_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 6, NEW.damage_resistance_physical_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 1, NEW.damage_resistance_energy_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 2, NEW.damage_resistance_energy_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 3, NEW.damage_resistance_energy_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 4, NEW.damage_resistance_energy_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 5, NEW.damage_resistance_energy_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 6, NEW.damage_resistance_energy_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 1, NEW.damage_resistance_radiation_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 2, NEW.damage_resistance_radiation_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 3, NEW.damage_resistance_radiation_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 4, NEW.damage_resistance_radiation_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 5, NEW.damage_resistance_radiation_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 6, NEW.damage_resistance_radiation_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS trg_sync_player_character_stats_after_insert
AFTER INSERT ON player_characters
BEGIN
    DELETE FROM player_character_defense_by_location
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_defense_by_location (player_character_id, body_location_id, defense, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.defense_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.defense_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.defense_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.defense_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 5, NEW.defense_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 6, NEW.defense_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM player_character_resistance_global
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_resistance_global (player_character_id, damage_type_id, resistance, immune, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.damage_resistance_physical, NEW.damage_resistance_physical_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.damage_resistance_energy, NEW.damage_resistance_energy_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.damage_resistance_radiation, NEW.damage_resistance_radiation_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.damage_resistance_poison, NEW.damage_resistance_poison_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM player_character_resistance_by_location
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_resistance_by_location (player_character_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
    VALUES
        (NEW.id, 1, 1, NEW.damage_resistance_physical_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 2, NEW.damage_resistance_physical_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 3, NEW.damage_resistance_physical_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 4, NEW.damage_resistance_physical_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 5, NEW.damage_resistance_physical_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 6, NEW.damage_resistance_physical_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 1, NEW.damage_resistance_energy_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 2, NEW.damage_resistance_energy_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 3, NEW.damage_resistance_energy_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 4, NEW.damage_resistance_energy_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 5, NEW.damage_resistance_energy_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 6, NEW.damage_resistance_energy_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 1, NEW.damage_resistance_radiation_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 2, NEW.damage_resistance_radiation_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 3, NEW.damage_resistance_radiation_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 4, NEW.damage_resistance_radiation_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 5, NEW.damage_resistance_radiation_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 6, NEW.damage_resistance_radiation_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS trg_sync_player_character_stats_after_update
AFTER UPDATE ON player_characters
BEGIN
    DELETE FROM player_character_defense_by_location
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_defense_by_location (player_character_id, body_location_id, defense, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.defense_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.defense_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.defense_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.defense_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 5, NEW.defense_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 6, NEW.defense_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM player_character_resistance_global
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_resistance_global (player_character_id, damage_type_id, resistance, immune, created_at, updated_at)
    VALUES
        (NEW.id, 1, NEW.damage_resistance_physical, NEW.damage_resistance_physical_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, NEW.damage_resistance_energy, NEW.damage_resistance_energy_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, NEW.damage_resistance_radiation, NEW.damage_resistance_radiation_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 4, NEW.damage_resistance_poison, NEW.damage_resistance_poison_immune, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

    DELETE FROM player_character_resistance_by_location
    WHERE player_character_id = NEW.id;

    INSERT INTO player_character_resistance_by_location (player_character_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
    VALUES
        (NEW.id, 1, 1, NEW.damage_resistance_physical_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 2, NEW.damage_resistance_physical_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 3, NEW.damage_resistance_physical_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 4, NEW.damage_resistance_physical_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 5, NEW.damage_resistance_physical_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 1, 6, NEW.damage_resistance_physical_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 1, NEW.damage_resistance_energy_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 2, NEW.damage_resistance_energy_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 3, NEW.damage_resistance_energy_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 4, NEW.damage_resistance_energy_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 5, NEW.damage_resistance_energy_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 2, 6, NEW.damage_resistance_energy_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 1, NEW.damage_resistance_radiation_head, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 2, NEW.damage_resistance_radiation_torso, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 3, NEW.damage_resistance_radiation_left_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 4, NEW.damage_resistance_radiation_right_arm, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 5, NEW.damage_resistance_radiation_left_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        (NEW.id, 3, 6, NEW.damage_resistance_radiation_right_leg, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'), STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_update;
DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_insert;
DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_update;
DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_insert;
