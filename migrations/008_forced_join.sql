CREATE TABLE forced_join_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_ref TEXT NOT NULL UNIQUE CHECK (length(chat_ref) BETWEEN 1 AND 128),
    display_name TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 128),
    join_url TEXT NOT NULL CHECK (length(join_url) BETWEEN 1 AND 512),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    required INTEGER NOT NULL DEFAULT 1 CHECK (required IN (0, 1)),
    position INTEGER NOT NULL DEFAULT 0 CHECK (position BETWEEN 0 AND 1000000),
    custom_text TEXT NOT NULL DEFAULT '' CHECK (length(custom_text) <= 1024),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_forced_join_channels_gate
    ON forced_join_channels(enabled, required, position, id);
