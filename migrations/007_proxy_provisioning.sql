CREATE TABLE proxy_user_provisioning (
    proxy_user_id INTEGER PRIMARY KEY REFERENCES proxy_users(id) ON DELETE CASCADE,
    phase TEXT NOT NULL CHECK (phase IN ('prepared', 'owned', 'collision')),
    secret_sha256 BLOB NOT NULL CHECK (length(secret_sha256) = 32),
    last_error_code TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (last_error_code IS NULL OR (length(last_error_code) BETWEEN 1 AND 64))
);

CREATE INDEX idx_proxy_user_provisioning_phase
    ON proxy_user_provisioning(phase, updated_at, proxy_user_id);
