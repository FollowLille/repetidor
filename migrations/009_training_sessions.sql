CREATE TABLE IF NOT EXISTS training_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mode TEXT NOT NULL,
    topic_id INTEGER,
    size INTEGER NOT NULL CHECK(size > 0),
    completed INTEGER NOT NULL DEFAULT 0,
    correct INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'completed')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS training_session_cards (
    session_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    word_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    direction TEXT NOT NULL,
    answer_mode TEXT NOT NULL,
    answered INTEGER NOT NULL DEFAULT 0,
    is_correct INTEGER NOT NULL DEFAULT 0,
    response TEXT NOT NULL DEFAULT '',
    answered_at DATETIME,
    PRIMARY KEY (session_id, position),
    FOREIGN KEY (session_id) REFERENCES training_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_training_sessions_status_updated
ON training_sessions(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_training_session_cards_word
ON training_session_cards(word_id);
