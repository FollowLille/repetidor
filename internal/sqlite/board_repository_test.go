package sqlite

import (
	"context"
	"path/filepath"
	"repetidor/internal/domain"
	"testing"
)

func TestBoardPersistsNodesPositionsAndEdges(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "boards.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	course, err := NewLearningCourseRepository(db).Create(ctx, domain.LearningCourse{LanguageTrackID: 1, Name: "Visual grammar"})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewBoardRepository(db)
	board, err := repo.Create(ctx, domain.Board{CourseID: course.ID, Name: "Pronouns"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.CreateNode(ctx, domain.BoardNode{BoardID: board.ID, Kind: "text", Title: "Yo", Content: "first person", X: 100, Y: 120, Width: 260, Height: 160, Color: "violet"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateNode(ctx, domain.BoardNode{BoardID: board.ID, Kind: "note", Title: "Estoy", Content: "estar", X: 500, Y: 300, Width: 260, Height: 160, Color: "amber"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.MoveNode(ctx, board.ID, first.ID, 222, 333); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateEdge(ctx, domain.BoardEdge{BoardID: board.ID, FromNodeID: first.ID, ToNodeID: second.ID, Label: "uses"}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, board.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 2 || len(got.Edges) != 1 || got.Nodes[0].X != 222 || got.Nodes[0].Y != 333 {
		t.Fatalf("board = %#v", got)
	}
}
