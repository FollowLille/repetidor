CREATE TABLE course_imports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    language_track_id INTEGER NOT NULL REFERENCES language_tracks(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    imported_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(language_track_id, fingerprint)
);

CREATE TABLE course_content_keys (
    language_track_id INTEGER NOT NULL REFERENCES language_tracks(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK(entity_type IN ('course','topic')),
    entity_id INTEGER NOT NULL,
    content_key TEXT NOT NULL,
    PRIMARY KEY(language_track_id, entity_type, content_key),
    UNIQUE(language_track_id, entity_type, entity_id)
);
