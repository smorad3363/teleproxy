CREATE TABLE proxy_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    desired_enabled INTEGER NOT NULL DEFAULT 1 CHECK (desired_enabled IN (0, 1)),
    sync_state TEXT NOT NULL DEFAULT 'pending' CHECK (sync_state IN ('pending', 'synced', 'error')),
    last_error_code TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_proxy_users_sync_state ON proxy_users(sync_state);
