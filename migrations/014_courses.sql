CREATE TABLE IF NOT EXISTS courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    target_language TEXT NOT NULL,
    reference_language TEXT NOT NULL,
    theory_language TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO courses(id, name, target_language, reference_language, theory_language)
SELECT 1, 'Испанский через русский', 'es', 'ru', 'ru'
WHERE NOT EXISTS (SELECT 1 FROM courses);

ALTER TABLE topics ADD COLUMN course_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS idx_topics_course_id ON topics(course_id);
