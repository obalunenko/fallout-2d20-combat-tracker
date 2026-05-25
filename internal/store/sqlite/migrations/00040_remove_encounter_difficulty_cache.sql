-- +goose Up
ALTER TABLE encounters DROP COLUMN difficulty_label;
ALTER TABLE encounters DROP COLUMN difficulty_score;
ALTER TABLE encounters DROP COLUMN party_count;
ALTER TABLE encounters DROP COLUMN party_avg_level;
ALTER TABLE encounters DROP COLUMN party_xp_budget;
ALTER TABLE encounters DROP COLUMN enemy_count;
ALTER TABLE encounters DROP COLUMN enemy_avg_level;
ALTER TABLE encounters DROP COLUMN enemy_total_xp;

-- +goose Down
ALTER TABLE encounters ADD COLUMN difficulty_label TEXT NOT NULL DEFAULT 'Unknown' CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly'));
ALTER TABLE encounters ADD COLUMN difficulty_score REAL NOT NULL DEFAULT 0 CHECK (difficulty_score >= 0);
ALTER TABLE encounters ADD COLUMN party_count INTEGER NOT NULL DEFAULT 0 CHECK (party_count >= 0);
ALTER TABLE encounters ADD COLUMN party_avg_level REAL NOT NULL DEFAULT 0 CHECK (party_avg_level >= 0);
ALTER TABLE encounters ADD COLUMN party_xp_budget INTEGER NOT NULL DEFAULT 0 CHECK (party_xp_budget >= 0);
ALTER TABLE encounters ADD COLUMN enemy_count INTEGER NOT NULL DEFAULT 0 CHECK (enemy_count >= 0);
ALTER TABLE encounters ADD COLUMN enemy_avg_level REAL NOT NULL DEFAULT 0 CHECK (enemy_avg_level >= 0);
ALTER TABLE encounters ADD COLUMN enemy_total_xp INTEGER NOT NULL DEFAULT 0 CHECK (enemy_total_xp >= 0);
