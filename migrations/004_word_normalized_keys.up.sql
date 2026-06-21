CREATE TABLE words_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_id INTEGER NOT NULL,
    spanish TEXT NOT NULL,
    spanish_key TEXT NOT NULL,
    russian TEXT NOT NULL,
    russian_key TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(topic_id, spanish_key, russian_key)
);

INSERT INTO words_new(id, topic_id, spanish, spanish_key, russian, russian_key, notes, created_at, updated_at)
SELECT id, topic_id, spanish, lower(trim(spanish)), russian, lower(trim(russian)), notes, created_at, updated_at
FROM words;

DROP TABLE words;
ALTER TABLE words_new RENAME TO words;
