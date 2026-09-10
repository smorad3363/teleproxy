CREATE TABLE bot_content (
    slot TEXT PRIMARY KEY CHECK (slot IN (
        'welcome',
        'forced_join',
        'referral',
        'proxy',
        'expired',
        'no_credit',
        'support'
    )),
    text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 4096),
    created_at INTEGER NOT NULL CHECK (created_at > 0),
    updated_at INTEGER NOT NULL CHECK (updated_at >= created_at)
);
