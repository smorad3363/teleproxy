CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (length(key) BETWEEN 1 AND 64),
    CHECK (length(value) BETWEEN 1 AND 1024)
);

CREATE TABLE telegram_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id INTEGER NOT NULL UNIQUE CHECK (telegram_id > 0),
    proxy_user_id INTEGER NOT NULL UNIQUE REFERENCES proxy_users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

ALTER TABLE credit_buckets
    ADD COLUMN idempotency_key TEXT
    CHECK (idempotency_key IS NULL OR length(idempotency_key) BETWEEN 1 AND 128);

CREATE UNIQUE INDEX idx_credit_buckets_user_idempotency
    ON credit_buckets(proxy_user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
