ALTER TABLE training_sessions ADD COLUMN abandoned_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_training_sessions_abandoned
ON training_sessions(abandoned_at);
