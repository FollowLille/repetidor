ALTER TABLE training_sessions ADD COLUMN direction_mode TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE training_sessions ADD COLUMN answer_mode TEXT NOT NULL DEFAULT 'auto';
