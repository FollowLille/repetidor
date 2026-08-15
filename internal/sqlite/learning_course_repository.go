package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"repetidor/internal/domain"
)

type LearningCourseRepository struct{ db *sql.DB }

func NewLearningCourseRepository(db *sql.DB) *LearningCourseRepository {
	return &LearningCourseRepository{db: db}
}

func (r *LearningCourseRepository) Get(ctx context.Context, id int64) (domain.LearningCourse, error) {
	var course domain.LearningCourse
	var parent sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT id,language_track_id,parent_id,name,description,sort_order,created_at,updated_at FROM courses WHERE id=?`, id).Scan(&course.ID, &course.LanguageTrackID, &parent, &course.Name, &course.Description, &course.SortOrder, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return course, fmt.Errorf("get learning course: %w", err)
	}
	if parent.Valid {
		course.ParentID = &parent.Int64
	}
	course.TopicIDs, err = r.ids(ctx, `SELECT topic_id FROM course_topics WHERE course_id=? ORDER BY sort_order,topic_id`, id)
	if err != nil {
		return course, err
	}
	course.PrerequisiteIDs, err = r.ids(ctx, `SELECT related_course_id FROM course_relations WHERE course_id=? AND relation_type='prerequisite' ORDER BY related_course_id`, id)
	return course, err
}

func (r *LearningCourseRepository) ListByTrack(ctx context.Context, trackID int64) ([]domain.LearningCourse, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,language_track_id,parent_id,name,description,sort_order,created_at,updated_at FROM courses WHERE language_track_id=? ORDER BY sort_order,id`, trackID)
	if err != nil {
		return nil, fmt.Errorf("list learning courses: %w", err)
	}
	defer rows.Close()
	var courses []domain.LearningCourse
	for rows.Next() {
		var course domain.LearningCourse
		var parent sql.NullInt64
		if err := rows.Scan(&course.ID, &course.LanguageTrackID, &parent, &course.Name, &course.Description, &course.SortOrder, &course.CreatedAt, &course.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan learning course: %w", err)
		}
		if parent.Valid {
			course.ParentID = &parent.Int64
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learning courses: %w", err)
	}
	for i := range courses {
		courses[i].TopicIDs, err = r.ids(ctx, `SELECT topic_id FROM course_topics WHERE course_id=? ORDER BY sort_order,topic_id`, courses[i].ID)
		if err != nil {
			return nil, err
		}
		courses[i].PrerequisiteIDs, err = r.ids(ctx, `SELECT related_course_id FROM course_relations WHERE course_id=? AND relation_type='prerequisite' ORDER BY related_course_id`, courses[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return courses, nil
}

func (r *LearningCourseRepository) ids(ctx context.Context, query string, id int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		ids = append(ids, value)
	}
	return ids, rows.Err()
}

func (r *LearningCourseRepository) Create(ctx context.Context, course domain.LearningCourse) (domain.LearningCourse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.LearningCourse{}, fmt.Errorf("begin learning course creation: %w", err)
	}
	defer tx.Rollback()
	var parent sql.NullInt64
	if course.ParentID != nil {
		parent = sql.NullInt64{Int64: *course.ParentID, Valid: true}
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO courses(language_track_id,parent_id,name,description,sort_order) VALUES(?,?,?,?,?) RETURNING id,created_at,updated_at`, course.LanguageTrackID, parent, course.Name, course.Description, course.SortOrder).Scan(&course.ID, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return domain.LearningCourse{}, fmt.Errorf("create learning course: %w", err)
	}
	for order, topicID := range course.TopicIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO course_topics(course_id,topic_id,sort_order) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM topics WHERE id=? AND language_track_id=?)`, course.ID, topicID, order, topicID, course.LanguageTrackID); err != nil {
			return domain.LearningCourse{}, fmt.Errorf("link course topic: %w", err)
		}
	}
	for _, prerequisiteID := range course.PrerequisiteIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO course_relations(course_id,related_course_id,relation_type) SELECT ?,?,'prerequisite' WHERE EXISTS(SELECT 1 FROM courses WHERE id=? AND language_track_id=?)`, course.ID, prerequisiteID, prerequisiteID, course.LanguageTrackID); err != nil {
			return domain.LearningCourse{}, fmt.Errorf("link course prerequisite: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.LearningCourse{}, fmt.Errorf("commit learning course creation: %w", err)
	}
	return course, nil
}

func (r *LearningCourseRepository) Update(ctx context.Context, course domain.LearningCourse) (domain.LearningCourse, error) {
	var parent sql.NullInt64
	if course.ParentID != nil {
		parent = sql.NullInt64{Int64: *course.ParentID, Valid: true}
	}
	err := r.db.QueryRowContext(ctx, `UPDATE courses SET parent_id=?,name=?,description=?,sort_order=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND language_track_id=? RETURNING created_at,updated_at`, parent, course.Name, course.Description, course.SortOrder, course.ID, course.LanguageTrackID).Scan(&course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return domain.LearningCourse{}, fmt.Errorf("update learning course: %w", err)
	}
	return course, nil
}

func (r *LearningCourseRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM courses WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete learning course: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("delete learning course: not found")
	}
	return nil
}
