CREATE TABLE board_strokes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('pen','arrow')),
    points TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'amber',
    width REAL NOT NULL DEFAULT 3,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_board_strokes_board ON board_strokes(board_id, id);
