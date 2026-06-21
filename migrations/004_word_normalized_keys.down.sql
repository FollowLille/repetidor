CREATE TABLE words_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_id INTEGER NOT NULL,
    spanish TEXT NOT NULL,
    russian TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(topic_id, spanish, russian)
);

INSERT INTO words_old(id, topic_id, spanish, russian, notes, created_at, updated_at)
SELECT id, topic_id, spanish, russian, notes, created_at, updated_at
FROM words;

DROP TABLE words;
ALTER TABLE words_old RENAME TO words;
