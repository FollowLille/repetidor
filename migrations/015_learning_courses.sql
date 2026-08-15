ALTER TABLE courses RENAME TO language_tracks;
ALTER TABLE topics RENAME COLUMN course_id TO language_track_id;

CREATE TABLE courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    language_track_id INTEGER NOT NULL REFERENCES language_tracks(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES courses(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(language_track_id, parent_id, name)
);

CREATE INDEX idx_courses_track_order ON courses(language_track_id, parent_id, sort_order, id);

CREATE TABLE course_topics (
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(course_id, topic_id)
);

CREATE INDEX idx_course_topics_topic ON course_topics(topic_id);

CREATE TABLE course_relations (
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    related_course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL CHECK(relation_type IN ('prerequisite', 'related')),
    PRIMARY KEY(course_id, related_course_id, relation_type),
    CHECK(course_id <> related_course_id)
);

