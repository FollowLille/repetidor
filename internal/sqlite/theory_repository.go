package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"repetidor/internal/domain"
)

type TheoryRepository struct{ db *sql.DB }

func NewTheoryRepository(db *sql.DB) *TheoryRepository { return &TheoryRepository{db: db} }

func (r *TheoryRepository) ListBlocks(ctx context.Context, courseID int64) ([]domain.TheoryBlock, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,kind,title,content,sort_order,created_at,updated_at FROM theory_blocks WHERE course_id=? ORDER BY sort_order,id`, courseID)
	if err != nil {
		return nil, fmt.Errorf("list theory blocks: %w", err)
	}
	defer rows.Close()
	var blocks []domain.TheoryBlock
	for rows.Next() {
		var b domain.TheoryBlock
		if err := rows.Scan(&b.ID, &b.CourseID, &b.Kind, &b.Title, &b.Content, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan theory block: %w", err)
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

func (r *TheoryRepository) CreateBlock(ctx context.Context, b domain.TheoryBlock) (domain.TheoryBlock, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO theory_blocks(course_id,kind,title,content,sort_order) VALUES(?,?,?,?,?) RETURNING id,created_at,updated_at`, b.CourseID, b.Kind, b.Title, b.Content, b.SortOrder).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return domain.TheoryBlock{}, fmt.Errorf("create theory block: %w", err)
	}
	return b, nil
}

func (r *TheoryRepository) DeleteBlock(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM theory_blocks WHERE id=?`, id)
	return err
}

func (r *TheoryRepository) ListExercises(ctx context.Context, courseID int64) ([]domain.TheoryExercise, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,theory_block_id,kind,prompt,options_json,correct_answer,accepted_answers_json,explanation,sort_order,created_at,updated_at FROM theory_exercises WHERE course_id=? ORDER BY sort_order,id`, courseID)
	if err != nil {
		return nil, fmt.Errorf("list theory exercises: %w", err)
	}
	defer rows.Close()
	var exercises []domain.TheoryExercise
	for rows.Next() {
		var e domain.TheoryExercise
		var block sql.NullInt64
		var options, accepted string
		if err := rows.Scan(&e.ID, &e.CourseID, &block, &e.Kind, &e.Prompt, &options, &e.CorrectAnswer, &accepted, &e.Explanation, &e.SortOrder, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan theory exercise: %w", err)
		}
		if block.Valid {
			e.TheoryBlockID = &block.Int64
		}
		_ = json.Unmarshal([]byte(options), &e.Options)
		_ = json.Unmarshal([]byte(accepted), &e.AcceptedAnswers)
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

func (r *TheoryRepository) CreateExercise(ctx context.Context, e domain.TheoryExercise) (domain.TheoryExercise, error) {
	options, _ := json.Marshal(e.Options)
	accepted, _ := json.Marshal(e.AcceptedAnswers)
	var block sql.NullInt64
	if e.TheoryBlockID != nil {
		block = sql.NullInt64{Int64: *e.TheoryBlockID, Valid: true}
	}
	err := r.db.QueryRowContext(ctx, `INSERT INTO theory_exercises(course_id,theory_block_id,kind,prompt,options_json,correct_answer,accepted_answers_json,explanation,sort_order) VALUES(?,?,?,?,?,?,?,?,?) RETURNING id,created_at,updated_at`, e.CourseID, block, e.Kind, e.Prompt, string(options), e.CorrectAnswer, string(accepted), e.Explanation, e.SortOrder).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return domain.TheoryExercise{}, fmt.Errorf("create theory exercise: %w", err)
	}
	return e, nil
}

func (r *TheoryRepository) DeleteExercise(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM theory_exercises WHERE id=?`, id)
	return err
}

func (r *TheoryRepository) Progress(ctx context.Context, courseID int64) (domain.CourseProgress, error) {
	var p domain.CourseProgress
	var read, complete sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT p.course_id,p.theory_read_at,p.practice_completed_at,p.practice_correct,p.practice_total,(SELECT COUNT(*) FROM theory_exercises e WHERE e.course_id=p.course_id),p.updated_at FROM course_progress p WHERE p.course_id=?`, courseID).Scan(&p.CourseID, &read, &complete, &p.PracticeCorrect, &p.PracticeTotal, &p.ExerciseCount, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		p.CourseID = courseID
		if countErr := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM theory_exercises WHERE course_id=?`, courseID).Scan(&p.ExerciseCount); countErr != nil {
			return p, fmt.Errorf("count theory exercises: %w", countErr)
		}
		return p, nil
	}
	if err != nil {
		return p, fmt.Errorf("get course progress: %w", err)
	}
	if read.Valid {
		p.TheoryReadAt = &read.Time
	}
	if complete.Valid {
		p.PracticeCompletedAt = &complete.Time
	}
	return p, nil
}

func (r *TheoryRepository) MarkTheoryRead(ctx context.Context, courseID int64) (domain.CourseProgress, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO course_progress(course_id,theory_read_at) VALUES(?,CURRENT_TIMESTAMP) ON CONFLICT(course_id) DO UPDATE SET theory_read_at=COALESCE(course_progress.theory_read_at,CURRENT_TIMESTAMP),updated_at=CURRENT_TIMESTAMP`, courseID)
	if err != nil {
		return domain.CourseProgress{}, fmt.Errorf("mark theory read: %w", err)
	}
	return r.Progress(ctx, courseID)
}

func (r *TheoryRepository) SubmitAnswer(ctx context.Context, exerciseID int64, answer string) (domain.TheoryAnswerResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.TheoryAnswerResult{}, err
	}
	defer tx.Rollback()
	var courseID int64
	var expected, acceptedJSON, explanation, language string
	if err := tx.QueryRowContext(ctx, `SELECT e.course_id,e.correct_answer,e.accepted_answers_json,e.explanation,lt.target_language FROM theory_exercises e JOIN courses c ON c.id=e.course_id JOIN language_tracks lt ON lt.id=c.language_track_id WHERE e.id=?`, exerciseID).Scan(&courseID, &expected, &acceptedJSON, &explanation, &language); err != nil {
		return domain.TheoryAnswerResult{}, fmt.Errorf("get theory exercise: %w", err)
	}
	var alternatives []string
	_ = json.Unmarshal([]byte(acceptedJSON), &alternatives)
	status := domain.CheckLanguageAnswer(language, answer, append([]string{expected}, alternatives...))
	correct := status != domain.TheoryAnswerWrong
	correctInt := 0
	if correct {
		correctInt = 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO theory_attempts(exercise_id,answer,is_correct) VALUES(?,?,?)`, exerciseID, strings.TrimSpace(answer), correctInt); err != nil {
		return domain.TheoryAnswerResult{}, fmt.Errorf("save theory attempt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO course_progress(course_id,practice_correct,practice_total) VALUES(?,?,1) ON CONFLICT(course_id) DO UPDATE SET practice_correct=course_progress.practice_correct+excluded.practice_correct,practice_total=course_progress.practice_total+1,updated_at=CURRENT_TIMESTAMP`, courseID, correctInt); err != nil {
		return domain.TheoryAnswerResult{}, fmt.Errorf("update theory progress: %w", err)
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM theory_exercises e WHERE e.course_id=? AND NOT EXISTS(SELECT 1 FROM theory_attempts a WHERE a.exercise_id=e.id)`, courseID).Scan(&remaining); err != nil {
		return domain.TheoryAnswerResult{}, err
	}
	if remaining == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE course_progress SET practice_completed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE course_id=?`, courseID); err != nil {
			return domain.TheoryAnswerResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.TheoryAnswerResult{}, err
	}
	progress, err := r.Progress(ctx, courseID)
	return domain.TheoryAnswerResult{Correct: correct, Status: status, Expected: expected, Explanation: explanation, Progress: progress}, err
}
