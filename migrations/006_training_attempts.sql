CREATE TABLE IF NOT EXISTS training_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word_id INTEGER NOT NULL,
    direction TEXT NOT NULL,
    question TEXT NOT NULL,
    expected TEXT NOT NULL,
    response TEXT NOT NULL,
    is_correct INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_training_attempts_word_id_created_at
ON training_attempts(word_id, created_at);
