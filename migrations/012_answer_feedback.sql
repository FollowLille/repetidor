ALTER TABLE training_session_cards ADD COLUMN error_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE training_session_cards ADD COLUMN edit_distance INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_training_session_cards_error_kind
ON training_session_cards(error_kind, word_id);
