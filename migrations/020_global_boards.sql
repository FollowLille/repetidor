PRAGMA defer_foreign_keys = ON;

ALTER TABLE board_edges RENAME TO old_board_edges;
ALTER TABLE board_strokes RENAME TO old_board_strokes;
ALTER TABLE board_nodes RENAME TO old_board_nodes;
ALTER TABLE boards RENAME TO old_boards;

DROP INDEX idx_boards_course;
DROP INDEX idx_board_nodes_board;
DROP INDEX idx_board_edges_board;
DROP INDEX idx_board_strokes_board;

CREATE TABLE boards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id INTEGER REFERENCES courses(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    background TEXT NOT NULL DEFAULT 'dots',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_boards_course ON boards(course_id, id);

CREATE TABLE board_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('text','note','image','audio')),
    title TEXT NOT NULL DEFAULT '', content TEXT NOT NULL DEFAULT '', media_path TEXT NOT NULL DEFAULT '',
    x REAL NOT NULL DEFAULT 80, y REAL NOT NULL DEFAULT 80, width REAL NOT NULL DEFAULT 260, height REAL NOT NULL DEFAULT 160,
    color TEXT NOT NULL DEFAULT 'violet', text_color TEXT NOT NULL DEFAULT 'white', z_index INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_board_nodes_board ON board_nodes(board_id, z_index, id);

CREATE TABLE board_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT, board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    from_node_id INTEGER NOT NULL REFERENCES board_nodes(id) ON DELETE CASCADE, to_node_id INTEGER NOT NULL REFERENCES board_nodes(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT '', color TEXT NOT NULL DEFAULT 'violet', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(board_id, from_node_id, to_node_id), CHECK(from_node_id <> to_node_id)
);
CREATE INDEX idx_board_edges_board ON board_edges(board_id, id);

CREATE TABLE board_strokes (
    id INTEGER PRIMARY KEY AUTOINCREMENT, board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('pen','arrow')), points TEXT NOT NULL, color TEXT NOT NULL DEFAULT 'amber', width REAL NOT NULL DEFAULT 3,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_board_strokes_board ON board_strokes(board_id, id);

INSERT INTO boards(id,course_id,name,description,background,created_at,updated_at)
SELECT id,course_id,name,description,'dots',created_at,updated_at FROM old_boards;
INSERT INTO board_nodes SELECT id,board_id,kind,title,content,media_path,x,y,width,height,color,text_color,z_index,created_at,updated_at FROM old_board_nodes;
INSERT INTO board_edges SELECT * FROM old_board_edges;
INSERT INTO board_strokes SELECT * FROM old_board_strokes;

DROP TABLE old_board_edges;
DROP TABLE old_board_strokes;
DROP TABLE old_board_nodes;
DROP TABLE old_boards;
