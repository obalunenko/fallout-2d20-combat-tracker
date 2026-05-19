-- +goose Up
ALTER TABLE encounters ADD COLUMN difficulty_label TEXT NOT NULL DEFAULT 'Unknown';
ALTER TABLE encounters ADD COLUMN difficulty_score REAL NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN party_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN party_avg_level REAL NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN party_xp_budget INTEGER NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN enemy_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN enemy_avg_level REAL NOT NULL DEFAULT 0;
ALTER TABLE encounters ADD COLUMN enemy_total_xp INTEGER NOT NULL DEFAULT 0;

WITH stats AS (
    SELECT
        e.id AS encounter_id,
        COALESCE(SUM(CASE WHEN c.side = 'party' THEN 1 ELSE 0 END), 0) AS party_count,
        COALESCE(AVG(CASE WHEN c.side = 'party' THEN c.level END), 0) AS party_avg_level,
        COALESCE(SUM(CASE WHEN c.side = 'npc' THEN 1 ELSE 0 END), 0) AS enemy_count,
        COALESCE(AVG(CASE WHEN c.side = 'npc' THEN c.level END), 0) AS enemy_avg_level,
        COALESCE(SUM(CASE WHEN c.side = 'npc' THEN c.xp ELSE 0 END), 0) AS enemy_total_xp
    FROM encounters e
    LEFT JOIN combatants c ON c.encounter_id = e.id
    GROUP BY e.id
),
calc AS (
    SELECT
        encounter_id,
        party_count,
        party_avg_level,
        CAST(ROUND((party_avg_level + 1) * party_count * 10, 0) AS INTEGER) AS party_xp_budget,
        enemy_count,
        enemy_avg_level,
        enemy_total_xp,
        CASE
            WHEN party_count = 0 OR enemy_count = 0 OR ((party_avg_level + 1) * party_count * 10) <= 0 THEN 0
            ELSE CAST(enemy_total_xp AS REAL) / ((party_avg_level + 1) * party_count * 10)
        END AS difficulty_score
    FROM stats
)
UPDATE encounters
SET
    party_count = (SELECT party_count FROM calc WHERE calc.encounter_id = encounters.id),
    party_avg_level = (SELECT party_avg_level FROM calc WHERE calc.encounter_id = encounters.id),
    party_xp_budget = (SELECT party_xp_budget FROM calc WHERE calc.encounter_id = encounters.id),
    enemy_count = (SELECT enemy_count FROM calc WHERE calc.encounter_id = encounters.id),
    enemy_avg_level = (SELECT enemy_avg_level FROM calc WHERE calc.encounter_id = encounters.id),
    enemy_total_xp = (SELECT enemy_total_xp FROM calc WHERE calc.encounter_id = encounters.id),
    difficulty_score = (SELECT ROUND(difficulty_score, 2) FROM calc WHERE calc.encounter_id = encounters.id),
    difficulty_label = (
        SELECT
            CASE
                WHEN party_count = 0 OR enemy_count = 0 THEN 'Unknown'
                WHEN difficulty_score < 0.5 THEN 'Trivial'
                WHEN difficulty_score < 1.0 THEN 'Easy'
                WHEN difficulty_score < 1.5 THEN 'Normal'
                WHEN difficulty_score <= 2.25 THEN 'Hard'
                ELSE 'Deadly'
            END
        FROM calc
        WHERE calc.encounter_id = encounters.id
    );

-- +goose Down
ALTER TABLE encounters DROP COLUMN enemy_total_xp;
ALTER TABLE encounters DROP COLUMN enemy_avg_level;
ALTER TABLE encounters DROP COLUMN enemy_count;
ALTER TABLE encounters DROP COLUMN party_xp_budget;
ALTER TABLE encounters DROP COLUMN party_avg_level;
ALTER TABLE encounters DROP COLUMN party_count;
ALTER TABLE encounters DROP COLUMN difficulty_score;
ALTER TABLE encounters DROP COLUMN difficulty_label;
