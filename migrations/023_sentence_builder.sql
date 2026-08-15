ALTER TABLE theory_attempts RENAME TO theory_attempts_legacy;
ALTER TABLE theory_exercises RENAME TO theory_exercises_legacy;

CREATE TABLE theory_exercises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    theory_block_id INTEGER REFERENCES theory_blocks(id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK(kind IN ('choice','input','gap','sentence_builder')),
    prompt TEXT NOT NULL,
    options_json TEXT NOT NULL DEFAULT '[]',
    correct_answer TEXT NOT NULL,
    accepted_answers_json TEXT NOT NULL DEFAULT '[]',
    explanation TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO theory_exercises SELECT id,course_id,theory_block_id,kind,prompt,options_json,correct_answer,accepted_answers_json,explanation,sort_order,created_at,updated_at FROM theory_exercises_legacy;

CREATE TABLE theory_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_id INTEGER NOT NULL REFERENCES theory_exercises(id) ON DELETE CASCADE,
    answer TEXT NOT NULL,
    is_correct INTEGER NOT NULL CHECK(is_correct IN (0,1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO theory_attempts SELECT id,exercise_id,answer,is_correct,created_at FROM theory_attempts_legacy;
DROP TABLE theory_attempts_legacy;
DROP TABLE theory_exercises_legacy;

CREATE INDEX idx_theory_exercises_course_order ON theory_exercises(course_id, sort_order, id);
CREATE INDEX idx_theory_attempts_exercise ON theory_attempts(exercise_id, created_at);
