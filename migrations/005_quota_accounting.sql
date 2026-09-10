ALTER TABLE quota_projection_members
ADD COLUMN accounted_bytes INTEGER NOT NULL DEFAULT 0
    CHECK (accounted_bytes >= 0 AND accounted_bytes <= allowance_bytes);
