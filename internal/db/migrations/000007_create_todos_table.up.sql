CREATE TABLE status_enum (
    name TEXT PRIMARY KEY,
    seq INTEGER
);

CREATE TABLE todos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER UNIQUE NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT ('pending') REFERENCES status_enum(name) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO status_enum (name, seq) VALUES ('pending', 1);
INSERT INTO status_enum (name, seq) VALUES ('done', 2);

INSERT INTO features (name, seq) VALUES ('todos', 2);
