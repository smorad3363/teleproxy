CREATE TABLE credit_buckets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    proxy_user_id INTEGER NOT NULL REFERENCES proxy_users(id) ON DELETE CASCADE,
    original_bytes INTEGER NOT NULL CHECK (original_bytes > 0),
    consumed_bytes INTEGER NOT NULL DEFAULT 0 CHECK (consumed_bytes >= 0 AND consumed_bytes <= original_bytes),
    starts_at INTEGER NOT NULL,
    expires_at INTEGER,
    reward_type TEXT NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (expires_at IS NULL OR expires_at > starts_at)
);

CREATE INDEX idx_credit_buckets_user_eligibility
    ON credit_buckets(proxy_user_id, status, expires_at, starts_at, id);
