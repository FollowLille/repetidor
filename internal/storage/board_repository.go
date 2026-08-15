package storage

import (
	"context"
	"repetidor/internal/domain"
)

type BoardRepository interface {
	ListByCourse(ctx context.Context, courseID int64) ([]domain.Board, error)
	Get(ctx context.Context, id int64) (domain.Board, error)
	Create(ctx context.Context, board domain.Board) (domain.Board, error)
	CreateNode(ctx context.Context, node domain.BoardNode) (domain.BoardNode, error)
	MoveNode(ctx context.Context, boardID, nodeID int64, x, y float64) error
	DeleteNode(ctx context.Context, boardID, nodeID int64) error
	CreateEdge(ctx context.Context, edge domain.BoardEdge) (domain.BoardEdge, error)
}
