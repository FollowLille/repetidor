CREATE TABLE theory_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('text','example','note','table')),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_theory_blocks_course_order ON theory_blocks(course_id, sort_order, id);

CREATE TABLE theory_exercises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    theory_block_id INTEGER REFERENCES theory_blocks(id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK(kind IN ('choice','input','gap')),
    prompt TEXT NOT NULL,
    options_json TEXT NOT NULL DEFAULT '[]',
    correct_answer TEXT NOT NULL,
    explanation TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_theory_exercises_course_order ON theory_exercises(course_id, sort_order, id);

CREATE TABLE course_progress (
    course_id INTEGER PRIMARY KEY REFERENCES courses(id) ON DELETE CASCADE,
    theory_read_at DATETIME,
    practice_completed_at DATETIME,
    practice_correct INTEGER NOT NULL DEFAULT 0,
    practice_total INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE theory_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_id INTEGER NOT NULL REFERENCES theory_exercises(id) ON DELETE CASCADE,
    answer TEXT NOT NULL,
    is_correct INTEGER NOT NULL CHECK(is_correct IN (0,1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_theory_attempts_exercise ON theory_attempts(exercise_id, created_at);

