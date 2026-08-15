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
	global, err := repo.Create(ctx, domain.Board{Name: "Language map", Background: "grid"})
	if err != nil {
		t.Fatal(err)
	}
	globals, err := repo.ListGlobal(ctx)
	if err != nil || len(globals) != 1 || globals[0].ID != global.ID || globals[0].Background != "grid" {
		t.Fatalf("global boards = %#v, err = %v", globals, err)
	}
	if err := repo.UpdateBackground(ctx, global.ID, "paper"); err != nil {
		t.Fatal(err)
	}
	board, err := repo.Create(ctx, domain.Board{CourseID: course.ID, Name: "Pronouns"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.CreateNode(ctx, domain.BoardNode{BoardID: board.ID, Kind: "text", Title: "Yo", Content: "first person", X: 100, Y: 120, Width: 260, Height: 160, Color: "violet", TextColor: "white"})
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
	if err := repo.ResizeNode(ctx, board.ID, first.ID, 420, 240); err != nil {
		t.Fatal(err)
	}
	first.Title, first.Content, first.Color, first.TextColor = "Yo estoy", "first person estar", "mint", "rose"
	if err := repo.UpdateNode(ctx, first); err != nil {
		t.Fatal(err)
	}
	edge, err := repo.CreateEdge(ctx, domain.BoardEdge{BoardID: board.ID, FromNodeID: first.ID, ToNodeID: second.ID, Label: "uses"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, board.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 2 || len(got.Edges) != 1 || got.Nodes[0].X != 222 || got.Nodes[0].Y != 333 || got.Nodes[0].Width != 420 || got.Nodes[0].Height != 240 || got.Nodes[0].Title != "Yo estoy" || got.Nodes[0].Color != "mint" || got.Nodes[0].TextColor != "rose" {
		t.Fatalf("board = %#v", got)
	}
	if err := repo.DeleteEdge(ctx, board.ID, edge.ID); err != nil {
		t.Fatal(err)
	}
	stroke, err := repo.CreateStroke(ctx, domain.BoardStroke{BoardID: board.ID, Kind: "pen", Points: `[{"x":10,"y":20},{"x":30,"y":40}]`, Color: "amber", Width: 3})
	if err != nil {
		t.Fatal(err)
	}
	withStroke, err := repo.Get(ctx, board.ID)
	if err != nil || len(withStroke.Strokes) != 1 || withStroke.Strokes[0].Points == "" {
		t.Fatalf("strokes = %#v, err = %v", withStroke.Strokes, err)
	}
	if err := repo.DeleteStroke(ctx, board.ID, stroke.ID); err != nil {
		t.Fatal(err)
	}
}
