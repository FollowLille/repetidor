CREATE TABLE IF NOT EXISTS word_topics (
    word_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (word_id, topic_id),
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

INSERT OR IGNORE INTO word_topics(word_id, topic_id)
SELECT id, topic_id FROM words;

-- Link every legacy duplicate to the canonical (oldest) word row.
INSERT OR IGNORE INTO word_topics(word_id, topic_id)
SELECT canonical.word_id, wt.topic_id
FROM word_topics wt
JOIN words duplicate_word ON duplicate_word.id = wt.word_id
JOIN (
    SELECT spanish_key, russian_key, MIN(id) AS word_id
    FROM words
    GROUP BY spanish_key, russian_key
) canonical
  ON canonical.spanish_key = duplicate_word.spanish_key
 AND canonical.russian_key = duplicate_word.russian_key;

-- Preserve attempts while redirecting them to the canonical word.
UPDATE training_attempts
SET word_id = (
    SELECT MIN(canonical.id)
    FROM words canonical
    JOIN words duplicate_word
      ON canonical.spanish_key = duplicate_word.spanish_key
     AND canonical.russian_key = duplicate_word.russian_key
    WHERE duplicate_word.id = training_attempts.word_id
);

-- Progress has a composite primary key, so aggregate before merging rows.
CREATE TEMP TABLE merged_training_progress AS
SELECT
    canonical.word_id AS word_id,
    progress.direction AS direction,
    SUM(progress.seen_count) AS seen_count,
    SUM(progress.correct_count) AS correct_count,
    SUM(progress.wrong_count) AS wrong_count,
    MAX(progress.correct_streak) AS correct_streak,
    MAX(progress.recent_pain) AS recent_pain,
    MAX(progress.last_seen_at) AS last_seen_at,
    MAX(progress.last_correct_at) AS last_correct_at,
    MAX(progress.updated_at) AS updated_at
FROM training_progress progress
JOIN words duplicate_word ON duplicate_word.id = progress.word_id
JOIN (
    SELECT spanish_key, russian_key, MIN(id) AS word_id
    FROM words
    GROUP BY spanish_key, russian_key
) canonical
  ON canonical.spanish_key = duplicate_word.spanish_key
 AND canonical.russian_key = duplicate_word.russian_key
GROUP BY canonical.word_id, progress.direction;

DELETE FROM training_progress;

INSERT INTO training_progress(
    word_id, direction, seen_count, correct_count, wrong_count,
    correct_streak, recent_pain, last_seen_at, last_correct_at, updated_at
)
SELECT
    word_id, direction, seen_count, correct_count, wrong_count,
    correct_streak, recent_pain, last_seen_at, last_correct_at, updated_at
FROM merged_training_progress;

DROP TABLE merged_training_progress;

DELETE FROM words
WHERE id NOT IN (
    SELECT MIN(id)
    FROM words
    GROUP BY spanish_key, russian_key
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_words_translation_key
ON words(spanish_key, russian_key);

CREATE INDEX IF NOT EXISTS idx_word_topics_topic_id_word_id
ON word_topics(topic_id, word_id);
