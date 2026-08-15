package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"repetidor/internal/domain"
)

type BoardRepository struct{ db *sql.DB }

func NewBoardRepository(db *sql.DB) *BoardRepository { return &BoardRepository{db: db} }

func (r *BoardRepository) ListByCourse(ctx context.Context, courseID int64) ([]domain.Board, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,name,description,created_at,updated_at FROM boards WHERE course_id=? ORDER BY id`, courseID)
	if err != nil {
		return nil, fmt.Errorf("list boards: %w", err)
	}
	defer rows.Close()
	var out []domain.Board
	for rows.Next() {
		var b domain.Board
		if err := rows.Scan(&b.ID, &b.CourseID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BoardRepository) Get(ctx context.Context, id int64) (domain.Board, error) {
	var b domain.Board
	err := r.db.QueryRowContext(ctx, `SELECT id,course_id,name,description,created_at,updated_at FROM boards WHERE id=?`, id).Scan(&b.ID, &b.CourseID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, fmt.Errorf("get board: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,board_id,kind,title,content,media_path,x,y,width,height,color,z_index,created_at,updated_at FROM board_nodes WHERE board_id=? ORDER BY z_index,id`, id)
	if err != nil {
		return b, err
	}
	for rows.Next() {
		var n domain.BoardNode
		if err := rows.Scan(&n.ID, &n.BoardID, &n.Kind, &n.Title, &n.Content, &n.MediaPath, &n.X, &n.Y, &n.Width, &n.Height, &n.Color, &n.ZIndex, &n.CreatedAt, &n.UpdatedAt); err != nil {
			rows.Close()
			return b, err
		}
		b.Nodes = append(b.Nodes, n)
	}
	rows.Close()
	edges, err := r.db.QueryContext(ctx, `SELECT id,board_id,from_node_id,to_node_id,label,color,created_at FROM board_edges WHERE board_id=? ORDER BY id`, id)
	if err != nil {
		return b, err
	}
	defer edges.Close()
	for edges.Next() {
		var e domain.BoardEdge
		if err := edges.Scan(&e.ID, &e.BoardID, &e.FromNodeID, &e.ToNodeID, &e.Label, &e.Color, &e.CreatedAt); err != nil {
			return b, err
		}
		b.Edges = append(b.Edges, e)
	}
	return b, edges.Err()
}

func (r *BoardRepository) Create(ctx context.Context, b domain.Board) (domain.Board, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO boards(course_id,name,description) VALUES(?,?,?) RETURNING id,created_at,updated_at`, b.CourseID, b.Name, b.Description).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, fmt.Errorf("create board: %w", err)
	}
	return b, nil
}
func (r *BoardRepository) CreateNode(ctx context.Context, n domain.BoardNode) (domain.BoardNode, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO board_nodes(board_id,kind,title,content,media_path,x,y,width,height,color,z_index) VALUES(?,?,?,?,?,?,?,?,?,?,?) RETURNING id,created_at,updated_at`, n.BoardID, n.Kind, n.Title, n.Content, n.MediaPath, n.X, n.Y, n.Width, n.Height, n.Color, n.ZIndex).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return n, fmt.Errorf("create board node: %w", err)
	}
	return n, nil
}
func (r *BoardRepository) MoveNode(ctx context.Context, boardID, nodeID int64, x, y float64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE board_nodes SET x=?,y=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND board_id=?`, x, y, nodeID, boardID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *BoardRepository) ResizeNode(ctx context.Context, boardID, nodeID int64, width, height float64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE board_nodes SET width=?,height=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND board_id=?`, width, height, nodeID, boardID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *BoardRepository) UpdateNode(ctx context.Context, n domain.BoardNode) error {
	result, err := r.db.ExecContext(ctx, `UPDATE board_nodes SET title=?,content=?,color=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND board_id=?`, n.Title, n.Content, n.Color, n.ID, n.BoardID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *BoardRepository) DeleteNode(ctx context.Context, boardID, nodeID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM board_nodes WHERE id=? AND board_id=?`, nodeID, boardID)
	return err
}
func (r *BoardRepository) CreateEdge(ctx context.Context, e domain.BoardEdge) (domain.BoardEdge, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO board_edges(board_id,from_node_id,to_node_id,label,color) SELECT ?,?,?,?,? WHERE EXISTS(SELECT 1 FROM board_nodes WHERE id=? AND board_id=?) AND EXISTS(SELECT 1 FROM board_nodes WHERE id=? AND board_id=?) RETURNING id,created_at`, e.BoardID, e.FromNodeID, e.ToNodeID, e.Label, e.Color, e.FromNodeID, e.BoardID, e.ToNodeID, e.BoardID).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return e, fmt.Errorf("create board edge: %w", err)
	}
	return e, nil
}
func (r *BoardRepository) DeleteEdge(ctx context.Context, boardID, edgeID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM board_edges WHERE id=? AND board_id=?`, edgeID, boardID)
	return err
}
