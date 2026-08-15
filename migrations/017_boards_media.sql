CREATE TABLE boards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_boards_course ON boards(course_id, id);

CREATE TABLE board_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('text','note','image','audio')),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    media_path TEXT NOT NULL DEFAULT '',
    x REAL NOT NULL DEFAULT 80,
    y REAL NOT NULL DEFAULT 80,
    width REAL NOT NULL DEFAULT 260,
    height REAL NOT NULL DEFAULT 160,
    color TEXT NOT NULL DEFAULT 'violet',
    z_index INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_board_nodes_board ON board_nodes(board_id, z_index, id);

CREATE TABLE board_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    from_node_id INTEGER NOT NULL REFERENCES board_nodes(id) ON DELETE CASCADE,
    to_node_id INTEGER NOT NULL REFERENCES board_nodes(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT 'violet',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(board_id, from_node_id, to_node_id),
    CHECK(from_node_id <> to_node_id)
);

CREATE INDEX idx_board_edges_board ON board_edges(board_id, id);

