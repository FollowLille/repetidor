CREATE TABLE IF NOT EXISTS training_progress (
    word_id INTEGER NOT NULL,
    direction TEXT NOT NULL,
    seen_count INTEGER NOT NULL DEFAULT 0,
    correct_count INTEGER NOT NULL DEFAULT 0,
    wrong_count INTEGER NOT NULL DEFAULT 0,
    correct_streak INTEGER NOT NULL DEFAULT 0,
    recent_pain INTEGER NOT NULL DEFAULT 0,
    last_seen_at DATETIME,
    last_correct_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (word_id, direction),
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_training_progress_direction_pain
ON training_progress(direction, recent_pain, correct_streak);
