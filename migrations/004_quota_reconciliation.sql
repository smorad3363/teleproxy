CREATE UNIQUE INDEX idx_credit_buckets_id_proxy_user
    ON credit_buckets(id, proxy_user_id);

CREATE TABLE quota_reconciliations (
    proxy_user_id INTEGER PRIMARY KEY REFERENCES proxy_users(id) ON DELETE CASCADE,
    generation INTEGER NOT NULL CHECK (generation > 0),
    phase TEXT NOT NULL CHECK (length(phase) BETWEEN 1 AND 32),
    telemt_reset_epoch_secs INTEGER NOT NULL CHECK (telemt_reset_epoch_secs >= 0),
    telemt_baseline_used_bytes INTEGER NOT NULL CHECK (telemt_baseline_used_bytes >= 0),
    projected_at INTEGER NOT NULL,
    projected_quota_bytes INTEGER NOT NULL CHECK (projected_quota_bytes >= 0),
    enforced_expiry INTEGER,
    next_start INTEGER,
    updated_at INTEGER NOT NULL,
    UNIQUE (proxy_user_id, generation)
);

CREATE TABLE quota_projection_members (
    proxy_user_id INTEGER NOT NULL,
    generation INTEGER NOT NULL CHECK (generation > 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    bucket_id INTEGER NOT NULL,
    allowance_bytes INTEGER NOT NULL CHECK (allowance_bytes > 0),
    PRIMARY KEY (proxy_user_id, generation, position),
    UNIQUE (proxy_user_id, generation, bucket_id),
    FOREIGN KEY (proxy_user_id, generation)
        REFERENCES quota_reconciliations(proxy_user_id, generation)
        ON DELETE CASCADE,
    FOREIGN KEY (bucket_id, proxy_user_id)
        REFERENCES credit_buckets(id, proxy_user_id)
        ON DELETE NO ACTION
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_quota_projection_members_bucket
    ON quota_projection_members(bucket_id);
