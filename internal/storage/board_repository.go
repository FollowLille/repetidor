package storage

import (
	"context"
	"repetidor/internal/domain"
)

type BoardRepository interface {
	ListByCourse(ctx context.Context, courseID int64) ([]domain.Board, error)
	ListGlobal(ctx context.Context) ([]domain.Board, error)
	Get(ctx context.Context, id int64) (domain.Board, error)
	Create(ctx context.Context, board domain.Board) (domain.Board, error)
	UpdateBackground(ctx context.Context, boardID int64, background string) error
	CreateNode(ctx context.Context, node domain.BoardNode) (domain.BoardNode, error)
	MoveNode(ctx context.Context, boardID, nodeID int64, x, y float64) error
	ResizeNode(ctx context.Context, boardID, nodeID int64, width, height float64) error
	UpdateNode(ctx context.Context, node domain.BoardNode) error
	DeleteNode(ctx context.Context, boardID, nodeID int64) error
	CreateEdge(ctx context.Context, edge domain.BoardEdge) (domain.BoardEdge, error)
	DeleteEdge(ctx context.Context, boardID, edgeID int64) error
	CreateStroke(ctx context.Context, stroke domain.BoardStroke) (domain.BoardStroke, error)
	DeleteStroke(ctx context.Context, boardID, strokeID int64) error
}
