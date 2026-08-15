ALTER TABLE course_content_keys RENAME TO course_content_keys_legacy;

CREATE TABLE course_content_keys (
    language_track_id INTEGER NOT NULL REFERENCES language_tracks(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK(entity_type IN ('course','topic','theory_block','theory_exercise','vocabulary')),
    entity_id INTEGER NOT NULL,
    content_key TEXT NOT NULL,
    PRIMARY KEY(language_track_id, entity_type, content_key)
);

INSERT INTO course_content_keys(language_track_id,entity_type,entity_id,content_key)
SELECT language_track_id,entity_type,entity_id,content_key FROM course_content_keys_legacy;

DROP TABLE course_content_keys_legacy;
CREATE INDEX idx_course_content_entity ON course_content_keys(language_track_id,entity_type,entity_id);
