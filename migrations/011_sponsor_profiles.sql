CREATE TABLE sponsor_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 128),
    channel_ref TEXT NOT NULL CHECK (length(channel_ref) BETWEEN 1 AND 512),
    ad_tag TEXT NOT NULL UNIQUE CHECK (
        length(ad_tag) = 32
        AND ad_tag NOT GLOB '*[^0-9a-f]*'
    ),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    weight INTEGER NOT NULL CHECK (weight > 0),
    starts_at INTEGER,
    ends_at INTEGER,
    notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 4096),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (starts_at IS NULL OR ends_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX idx_sponsor_profiles_active
    ON sponsor_profiles(enabled, starts_at, ends_at, id);
