CREATE TABLE bot_runtime_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    username TEXT NOT NULL CHECK (length(username) BETWEEN 1 AND 64),
    admin_chat_id INTEGER NOT NULL CHECK (admin_chat_id > 0),
    enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    created_at INTEGER NOT NULL CHECK (created_at > 0),
    updated_at INTEGER NOT NULL CHECK (updated_at >= created_at)
);
