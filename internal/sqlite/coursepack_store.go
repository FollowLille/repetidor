package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"repetidor/internal/coursepack"
	"repetidor/internal/storage"
)

var ErrCoursePackageDuplicate = storage.ErrCoursePackageDuplicate

type CoursePackageSummary = storage.CoursePackageSummary

type CoursePackageStore struct{ db *sql.DB }

func NewCoursePackageStore(db *sql.DB) *CoursePackageStore { return &CoursePackageStore{db: db} }

func (s *CoursePackageStore) Export(ctx context.Context, courseID int64) (coursepack.Package, error) {
	value := coursepack.Package{Format: coursepack.Format, Version: coursepack.Version}
	var parent sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT c.name,c.description,c.sort_order,c.parent_id,lt.target_language,lt.reference_language,lt.theory_language,COALESCE(k.content_key,'course-'||c.id) FROM courses c JOIN language_tracks lt ON lt.id=c.language_track_id LEFT JOIN course_content_keys k ON k.language_track_id=c.language_track_id AND k.entity_type='course' AND k.entity_id=c.id WHERE c.id=?`, courseID).Scan(&value.Course.Name, &value.Course.Description, &value.Course.SortOrder, &parent, &value.Course.Target, &value.Course.Reference, &value.Course.Theory, &value.Course.Key)
	if err != nil {
		return value, fmt.Errorf("export course metadata: %w", err)
	}
	if parent.Valid {
		_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(content_key,'') FROM course_content_keys WHERE entity_type='course' AND entity_id=?`, parent.Int64).Scan(&value.Course.ParentKey)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT l.code FROM learning_levels l JOIN learning_course_levels cl ON cl.level_id=l.id WHERE cl.course_id=? ORDER BY l.sort_order,l.id`, courseID)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			rows.Close()
			return value, err
		}
		value.Course.Levels = append(value.Course.Levels, code)
	}
	rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT COALESCE(k.content_key,''),r.related_course_id FROM course_relations r LEFT JOIN courses rc ON rc.id=r.related_course_id LEFT JOIN course_content_keys k ON k.language_track_id=rc.language_track_id AND k.entity_type='course' AND k.entity_id=rc.id WHERE r.course_id=? AND r.relation_type='prerequisite' ORDER BY r.related_course_id`, courseID)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var key string
		var ignored int64
		if err = rows.Scan(&key, &ignored); err != nil {
			rows.Close()
			return value, err
		}
		if key != "" {
			value.Course.Prerequisites = append(value.Course.Prerequisites, key)
		}
	}
	rows.Close()
	blockKeys := map[int64]string{}
	rows, err = s.db.QueryContext(ctx, `SELECT b.id,COALESCE((SELECT MIN(content_key) FROM course_content_keys k WHERE k.entity_type='theory_block' AND k.entity_id=b.id),'block-'||b.id),b.kind,b.title,b.content,b.sort_order FROM theory_blocks b WHERE b.course_id=? ORDER BY b.sort_order,b.id`, courseID)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var id int64
		var block coursepack.Block
		if err = rows.Scan(&id, &block.Key, &block.Kind, &block.Title, &block.Content, &block.SortOrder); err != nil {
			rows.Close()
			return value, err
		}
		blockKeys[id] = block.Key
		value.Course.Blocks = append(value.Course.Blocks, block)
	}
	rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT e.id,COALESCE((SELECT MIN(content_key) FROM course_content_keys k WHERE k.entity_type='theory_exercise' AND k.entity_id=e.id),'exercise-'||e.id),e.theory_block_id,e.kind,e.prompt,e.options_json,e.correct_answer,e.accepted_answers_json,e.explanation,e.sort_order FROM theory_exercises e WHERE e.course_id=? ORDER BY e.sort_order,e.id`, courseID)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var id int64
		var block sql.NullInt64
		var options, accepted string
		var exercise coursepack.Exercise
		if err = rows.Scan(&id, &exercise.Key, &block, &exercise.Kind, &exercise.Prompt, &options, &exercise.CorrectAnswer, &accepted, &exercise.Explanation, &exercise.SortOrder); err != nil {
			rows.Close()
			return value, err
		}
		if block.Valid {
			exercise.BlockKey = blockKeys[block.Int64]
		}
		_ = json.Unmarshal([]byte(options), &exercise.Options)
		_ = json.Unmarshal([]byte(accepted), &exercise.AcceptedAnswers)
		value.Course.Exercises = append(value.Course.Exercises, exercise)
	}
	rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT t.id,COALESCE((SELECT MIN(content_key) FROM course_content_keys k WHERE k.language_track_id=t.language_track_id AND k.entity_type='topic' AND k.entity_id=t.id),'topic-'||t.id),t.name,t.description,ct.sort_order FROM course_topics ct JOIN topics t ON t.id=ct.topic_id WHERE ct.course_id=? ORDER BY ct.sort_order,t.id`, courseID)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var topicID int64
		var topic coursepack.Topic
		if err = rows.Scan(&topicID, &topic.Key, &topic.Name, &topic.Description, &topic.SortOrder); err != nil {
			rows.Close()
			return value, err
		}
		wordRows, wordErr := s.db.QueryContext(ctx, `SELECT COALESCE((SELECT MIN(content_key) FROM course_content_keys k WHERE k.entity_type='vocabulary' AND k.entity_id=w.id),''),w.spanish,w.russian,w.notes FROM words w JOIN word_topics wt ON wt.word_id=w.id WHERE wt.topic_id=? ORDER BY w.spanish_key,w.russian_key`, topicID)
		if wordErr != nil {
			rows.Close()
			return value, wordErr
		}
		for wordRows.Next() {
			var word coursepack.Word
			if wordErr = wordRows.Scan(&word.Key, &word.Target, &word.Reference, &word.Notes); wordErr != nil {
				wordRows.Close()
				rows.Close()
				return value, wordErr
			}
			topic.Words = append(topic.Words, word)
		}
		wordRows.Close()
		value.Course.Topics = append(value.Course.Topics, topic)
	}
	rows.Close()
	return value, nil
}

func packageFingerprint(value coursepack.Package) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func summarize(value coursepack.Package) CoursePackageSummary {
	result := CoursePackageSummary{Blocks: len(value.Course.Blocks), Exercises: len(value.Course.Exercises), Topics: len(value.Course.Topics)}
	for _, topic := range value.Course.Topics {
		result.Words += len(topic.Words)
	}
	return result
}

func (s *CoursePackageStore) Preview(ctx context.Context, trackID int64, value coursepack.Package) (CoursePackageSummary, error) {
	if err := value.Validate(); err != nil {
		return CoursePackageSummary{}, err
	}
	result := summarize(value)
	var target, reference, theory string
	if err := s.db.QueryRowContext(ctx, `SELECT target_language,reference_language,theory_language FROM language_tracks WHERE id=?`, trackID).Scan(&target, &reference, &theory); err != nil {
		return result, fmt.Errorf("language track not found: %w", err)
	}
	if target != value.Course.Target || reference != value.Course.Reference {
		return result, fmt.Errorf("course languages %s/%s do not match the active track %s/%s", value.Course.Target, value.Course.Reference, target, reference)
	}
	if value.Course.Theory != "" && value.Course.Theory != theory {
		return result, fmt.Errorf("course theory language %s does not match the active track %s", value.Course.Theory, theory)
	}
	for _, raw := range value.Course.Levels {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM learning_levels WHERE track_id=? AND code=?`, trackID, normalizeLevelCode(raw)).Scan(&exists)
		if err == sql.ErrNoRows {
			return result, fmt.Errorf("unknown learning level %q for active language track", raw)
		}
		if err != nil {
			return result, err
		}
	}
	fingerprint, err := packageFingerprint(value)
	if err != nil {
		return result, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT course_id FROM course_imports WHERE language_track_id=? AND fingerprint=?`, trackID, fingerprint).Scan(&result.DuplicateCourseID)
	if err != nil && err != sql.ErrNoRows {
		return result, fmt.Errorf("preview course package: %w", err)
	}
	if result.DuplicateCourseID > 0 {
		result.Duplicates = 1
		return result, nil
	}
	for kind, key := range map[string]string{"course": value.Course.Key} {
		var exists int
		err = s.db.QueryRowContext(ctx, `SELECT 1 FROM course_content_keys WHERE language_track_id=? AND entity_type=? AND content_key=?`, trackID, kind, key).Scan(&exists)
		if err == nil {
			result.Duplicates++
		} else if err != sql.ErrNoRows {
			return result, err
		}
	}
	for _, block := range value.Course.Blocks {
		var exists int
		err = s.db.QueryRowContext(ctx, `SELECT 1 FROM course_content_keys WHERE language_track_id=? AND entity_type='theory_block' AND content_key=?`, trackID, block.Key).Scan(&exists)
		if err == nil {
			result.Duplicates++
		} else if err != sql.ErrNoRows {
			return result, err
		}
	}
	for _, exercise := range value.Course.Exercises {
		var exists int
		err = s.db.QueryRowContext(ctx, `SELECT 1 FROM course_content_keys WHERE language_track_id=? AND entity_type='theory_exercise' AND content_key=?`, trackID, exercise.Key).Scan(&exists)
		if err == nil {
			result.Duplicates++
		} else if err != sql.ErrNoRows {
			return result, err
		}
	}
	for _, topic := range value.Course.Topics {
		var exists int
		err = s.db.QueryRowContext(ctx, `SELECT 1 FROM topics WHERE language_track_id=? AND lower(name)=lower(?)`, trackID, topic.Name).Scan(&exists)
		if err == nil {
			result.Duplicates++
		} else if err != sql.ErrNoRows {
			return result, err
		}
		for _, word := range topic.Words {
			err = s.db.QueryRowContext(ctx, `SELECT 1 FROM words WHERE spanish_key=? AND russian_key=? LIMIT 1`, normalizeWordKey(word.Target), normalizeWordKey(word.Reference)).Scan(&exists)
			if err == nil {
				result.Duplicates++
			} else if err != sql.ErrNoRows {
				return result, err
			}
		}
	}
	return result, nil
}

func (s *CoursePackageStore) Import(ctx context.Context, trackID int64, value coursepack.Package) (int64, CoursePackageSummary, error) {
	if err := value.Validate(); err != nil {
		return 0, CoursePackageSummary{}, err
	}
	result := summarize(value)
	fingerprint, err := packageFingerprint(value)
	if err != nil {
		return 0, result, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, result, err
	}
	defer tx.Rollback()
	var target, reference, theory string
	if err = tx.QueryRowContext(ctx, `SELECT target_language,reference_language,theory_language FROM language_tracks WHERE id=?`, trackID).Scan(&target, &reference, &theory); err != nil {
		return 0, result, fmt.Errorf("language track not found: %w", err)
	}
	if target != value.Course.Target || reference != value.Course.Reference {
		return 0, result, fmt.Errorf("course languages %s/%s do not match the active track %s/%s", value.Course.Target, value.Course.Reference, target, reference)
	}
	if value.Course.Theory != "" && value.Course.Theory != theory {
		return 0, result, fmt.Errorf("course theory language %s does not match the active track %s", value.Course.Theory, theory)
	}
	if err = validatePackageLevels(ctx, tx, trackID, value.Course.Levels); err != nil {
		return 0, result, err
	}
	var duplicateID int64
	if err = tx.QueryRowContext(ctx, `SELECT course_id FROM course_imports WHERE language_track_id=? AND fingerprint=?`, trackID, fingerprint).Scan(&duplicateID); err == nil {
		result.Duplicates = 1
		result.DuplicateCourseID = duplicateID
		return duplicateID, result, ErrCoursePackageDuplicate
	} else if err != sql.ErrNoRows {
		return 0, result, err
	}
	parentID, err := resolveContentKey(ctx, tx, trackID, "course", value.Course.ParentKey)
	if err != nil {
		return 0, result, err
	}
	var parent sql.NullInt64
	if parentID > 0 {
		parent = sql.NullInt64{Int64: parentID, Valid: true}
	}
	var courseID int64
	courseID, err = findContentKey(ctx, tx, trackID, "course", value.Course.Key)
	if err != nil {
		return 0, result, err
	}
	if courseID > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE courses SET parent_id=?,name=?,description=?,sort_order=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND language_track_id=?`, parent, value.Course.Name, value.Course.Description, value.Course.SortOrder, courseID, trackID); err != nil {
			return 0, result, friendlyImportError("update course", err)
		}
	} else {
		if err = tx.QueryRowContext(ctx, `INSERT INTO courses(language_track_id,parent_id,name,description,sort_order) VALUES(?,?,?,?,?) RETURNING id`, trackID, parent, value.Course.Name, value.Course.Description, value.Course.SortOrder).Scan(&courseID); err != nil {
			return 0, result, friendlyImportError("create course", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO course_content_keys(language_track_id,entity_type,entity_id,content_key) VALUES(?,'course',?,?)`, trackID, courseID, value.Course.Key); err != nil {
			return 0, result, err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM learning_course_levels WHERE course_id=?`, courseID); err != nil {
		return 0, result, err
	}
	for _, code := range value.Course.Levels {
		if _, err = tx.ExecContext(ctx, `INSERT INTO learning_course_levels(course_id,level_id) SELECT ?,id FROM learning_levels WHERE track_id=? AND code=?`, courseID, trackID, normalizeLevelCode(code)); err != nil {
			return 0, result, err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM course_relations WHERE course_id=? AND relation_type='prerequisite'`, courseID); err != nil {
		return 0, result, err
	}
	for _, key := range value.Course.Prerequisites {
		id, resolveErr := resolveContentKey(ctx, tx, trackID, "course", key)
		if resolveErr != nil {
			return 0, result, resolveErr
		}
		if id > 0 {
			if _, err = tx.ExecContext(ctx, `INSERT INTO course_relations(course_id,related_course_id,relation_type) VALUES(?,?,'prerequisite')`, courseID, id); err != nil {
				return 0, result, err
			}
		}
	}
	blockIDs := map[string]int64{}
	for _, block := range value.Course.Blocks {
		id, findErr := findContentKey(ctx, tx, trackID, "theory_block", block.Key)
		if findErr != nil {
			return 0, result, findErr
		}
		if id > 0 {
			res, updateErr := tx.ExecContext(ctx, `UPDATE theory_blocks SET kind=?,title=?,content=?,sort_order=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND course_id=?`, block.Kind, block.Title, block.Content, block.SortOrder, id, courseID)
			if updateErr != nil {
				return 0, result, friendlyImportError("update theory block", updateErr)
			}
			if affected, _ := res.RowsAffected(); affected == 0 {
				return 0, result, fmt.Errorf("theory block key %q belongs to another course", block.Key)
			}
		} else {
			if err = tx.QueryRowContext(ctx, `INSERT INTO theory_blocks(course_id,kind,title,content,sort_order) VALUES(?,?,?,?,?) RETURNING id`, courseID, block.Kind, block.Title, block.Content, block.SortOrder).Scan(&id); err != nil {
				return 0, result, friendlyImportError("create theory block", err)
			}
			if err = registerContentKey(ctx, tx, trackID, "theory_block", id, block.Key); err != nil {
				return 0, result, err
			}
		}
		blockIDs[block.Key] = id
	}
	for _, exercise := range value.Course.Exercises {
		options, _ := json.Marshal(exercise.Options)
		accepted, _ := json.Marshal(exercise.AcceptedAnswers)
		var block sql.NullInt64
		if id := blockIDs[exercise.BlockKey]; id > 0 {
			block = sql.NullInt64{Int64: id, Valid: true}
		}
		id, findErr := findContentKey(ctx, tx, trackID, "theory_exercise", exercise.Key)
		if findErr != nil {
			return 0, result, findErr
		}
		if id > 0 {
			res, updateErr := tx.ExecContext(ctx, `UPDATE theory_exercises SET theory_block_id=?,kind=?,prompt=?,options_json=?,correct_answer=?,accepted_answers_json=?,explanation=?,sort_order=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND course_id=?`, block, exercise.Kind, exercise.Prompt, string(options), exercise.CorrectAnswer, string(accepted), exercise.Explanation, exercise.SortOrder, id, courseID)
			if updateErr != nil {
				return 0, result, friendlyImportError("update theory exercise", updateErr)
			}
			if affected, _ := res.RowsAffected(); affected == 0 {
				return 0, result, fmt.Errorf("theory exercise key %q belongs to another course", exercise.Key)
			}
		} else {
			if err = tx.QueryRowContext(ctx, `INSERT INTO theory_exercises(course_id,theory_block_id,kind,prompt,options_json,correct_answer,accepted_answers_json,explanation,sort_order) VALUES(?,?,?,?,?,?,?,?,?) RETURNING id`, courseID, block, exercise.Kind, exercise.Prompt, string(options), exercise.CorrectAnswer, string(accepted), exercise.Explanation, exercise.SortOrder).Scan(&id); err != nil {
				return 0, result, friendlyImportError("create theory exercise", err)
			}
			if err = registerContentKey(ctx, tx, trackID, "theory_exercise", id, exercise.Key); err != nil {
				return 0, result, err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM course_topics WHERE course_id=?`, courseID); err != nil {
		return 0, result, err
	}
	for order, topic := range value.Course.Topics {
		topicID, created, topicErr := upsertPackageTopic(ctx, tx, trackID, topic)
		if topicErr != nil {
			return 0, result, topicErr
		}
		if !created {
			result.Duplicates++
		}
		topicOrder := topic.SortOrder
		if topicOrder == 0 {
			topicOrder = order
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO course_topics(course_id,topic_id,sort_order) VALUES(?,?,?)`, courseID, topicID, topicOrder); err != nil {
			return 0, result, err
		}
		for _, word := range topic.Words {
			wordID, wordCreated, wordErr := upsertPackageWord(ctx, tx, trackID, topicID, word)
			if wordErr != nil {
				return 0, result, wordErr
			}
			if !wordCreated {
				result.Duplicates++
			}
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO word_topics(word_id,topic_id) VALUES(?,?)`, wordID, topicID); err != nil {
				return 0, result, err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM course_imports WHERE language_track_id=? AND course_id=?`, trackID, courseID); err != nil {
		return 0, result, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO course_imports(language_track_id,fingerprint,course_id) VALUES(?,?,?)`, trackID, fingerprint, courseID); err != nil {
		return 0, result, err
	}
	if err = tx.Commit(); err != nil {
		return 0, result, err
	}
	return courseID, result, nil
}

func resolveContentKey(ctx context.Context, tx *sql.Tx, trackID int64, kind, key string) (int64, error) {
	if key == "" {
		return 0, nil
	}
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT entity_id FROM course_content_keys WHERE language_track_id=? AND entity_type=? AND content_key=?`, trackID, kind, key).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("unknown %s key %q", kind, key)
	}
	return id, err
}

func findContentKey(ctx context.Context, tx *sql.Tx, trackID int64, kind, key string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT entity_id FROM course_content_keys WHERE language_track_id=? AND entity_type=? AND content_key=?`, trackID, kind, key).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}
func registerContentKey(ctx context.Context, tx *sql.Tx, trackID int64, kind string, id int64, key string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO course_content_keys(language_track_id,entity_type,entity_id,content_key) VALUES(?,?,?,?)`, trackID, kind, id, key)
	return err
}
func validatePackageLevels(ctx context.Context, tx *sql.Tx, trackID int64, codes []string) error {
	for _, raw := range codes {
		code := normalizeLevelCode(raw)
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM learning_levels WHERE track_id=? AND code=?`, trackID, code).Scan(&exists)
		if err == sql.ErrNoRows {
			return fmt.Errorf("unknown learning level %q for active language track", raw)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeLevelCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func upsertPackageTopic(ctx context.Context, tx *sql.Tx, trackID int64, topic coursepack.Topic) (int64, bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT entity_id FROM course_content_keys WHERE language_track_id=? AND entity_type='topic' AND content_key=?`, trackID, topic.Key).Scan(&id)
	if err == nil {
		if _, updateErr := tx.ExecContext(ctx, `UPDATE topics SET name=?,description=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND language_track_id=?`, topic.Name, topic.Description, id, trackID); updateErr != nil {
			return 0, false, friendlyImportError("update topic", updateErr)
		}
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}
	err = tx.QueryRowContext(ctx, `SELECT id FROM topics WHERE language_track_id=? AND lower(name)=lower(?)`, trackID, topic.Name).Scan(&id)
	created := false
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `INSERT INTO topics(language_track_id,name,description) VALUES(?,?,?) RETURNING id`, trackID, topic.Name, topic.Description).Scan(&id)
		created = true
	} else if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE topics SET name=?,description=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, topic.Name, topic.Description, id)
	}
	if err != nil {
		return 0, false, friendlyImportError("create topic", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO course_content_keys(language_track_id,entity_type,entity_id,content_key) VALUES(?,'topic',?,?)`, trackID, id, topic.Key)
	return id, created, err
}

func upsertPackageWord(ctx context.Context, tx *sql.Tx, trackID, topicID int64, word coursepack.Word) (int64, bool, error) {
	targetKey, referenceKey := normalizeWordKey(word.Target), normalizeWordKey(word.Reference)
	var id int64
	if word.Key != "" {
		var keyErr error
		id, keyErr = findContentKey(ctx, tx, trackID, "vocabulary", word.Key)
		if keyErr != nil {
			return 0, false, keyErr
		}
		if id > 0 {
			if _, updateErr := tx.ExecContext(ctx, `UPDATE words SET spanish=?,spanish_key=?,russian=?,russian_key=?,notes=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, word.Target, targetKey, word.Reference, referenceKey, word.Notes, id); updateErr != nil {
				return 0, false, friendlyImportError("update vocabulary", updateErr)
			}
			return id, false, nil
		}
	}
	err := tx.QueryRowContext(ctx, `SELECT id FROM words WHERE spanish_key=? AND russian_key=? ORDER BY id LIMIT 1`, targetKey, referenceKey).Scan(&id)
	if err == nil {
		if word.Key != "" {
			if keyErr := registerContentKey(ctx, tx, trackID, "vocabulary", id, word.Key); keyErr != nil {
				return 0, false, keyErr
			}
		}
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO words(topic_id,spanish,spanish_key,russian,russian_key,notes) VALUES(?,?,?,?,?,?) RETURNING id`, topicID, word.Target, targetKey, word.Reference, referenceKey, word.Notes).Scan(&id)
	if err != nil {
		return 0, false, friendlyImportError("create vocabulary", err)
	}
	if word.Key != "" {
		if err = registerContentKey(ctx, tx, trackID, "vocabulary", id, word.Key); err != nil {
			return 0, false, err
		}
	}
	return id, true, nil
}

func friendlyImportError(action string, err error) error {
	if err == nil {
		return nil
	}
	if isUniqueConstraintError(err) {
		return fmt.Errorf("%s: content already exists", action)
	}
	if strings.Contains(strings.ToLower(err.Error()), "check constraint") {
		return fmt.Errorf("%s: unsupported content type", action)
	}
	return fmt.Errorf("%s: %w", action, err)
}
