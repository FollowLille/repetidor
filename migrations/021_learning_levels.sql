CREATE TABLE learning_levels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id INTEGER NOT NULL REFERENCES language_tracks(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE(track_id, code)
);

CREATE TABLE learning_course_levels (
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES learning_levels(id) ON DELETE CASCADE,
    PRIMARY KEY(course_id, level_id)
);

INSERT INTO learning_levels(track_id,code,name,sort_order)
SELECT id,'A1','A1 · Beginner',10 FROM language_tracks UNION ALL
SELECT id,'A2','A2 · Elementary',20 FROM language_tracks UNION ALL
SELECT id,'B1','B1 · Intermediate',30 FROM language_tracks UNION ALL
SELECT id,'B2','B2 · Upper intermediate',40 FROM language_tracks UNION ALL
SELECT id,'C1','C1 · Advanced',50 FROM language_tracks UNION ALL
SELECT id,'C2','C2 · Proficiency',60 FROM language_tracks;
