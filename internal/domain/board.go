package domain

import "time"

type Board struct {
	ID                            int64
	CourseID                      int64
	Name, Description, Background string
	Nodes                         []BoardNode
	Edges                         []BoardEdge
	Strokes                       []BoardStroke
	CreatedAt, UpdatedAt          time.Time
}

type BoardStroke struct {
	ID, BoardID         int64
	Kind, Points, Color string
	Width               float64
	CreatedAt           time.Time
}

type BoardNode struct {
	ID, BoardID                     int64
	Kind, Title, Content, MediaPath string
	X, Y, Width, Height             float64
	Color, TextColor                string
	ZIndex                          int
	CreatedAt, UpdatedAt            time.Time
}

type BoardEdge struct {
	ID, BoardID, FromNodeID, ToNodeID int64
	Label, Color                      string
	CreatedAt                         time.Time
}
