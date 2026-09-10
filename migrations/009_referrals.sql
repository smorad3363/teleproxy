CREATE TABLE referral_codes (
    telegram_user_id INTEGER PRIMARY KEY REFERENCES telegram_users(id) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE CHECK (length(code) BETWEEN 16 AND 64),
    created_at INTEGER NOT NULL
);

CREATE TABLE referral_attributions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inviter_user_id INTEGER NOT NULL REFERENCES telegram_users(id) ON DELETE CASCADE,
    invitee_user_id INTEGER NOT NULL UNIQUE REFERENCES telegram_users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'rewarded', 'rejected')),
    rejection_reason TEXT CHECK (rejection_reason IS NULL OR length(rejection_reason) BETWEEN 1 AND 128),
    finalized_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (inviter_user_id <> invitee_user_id),
    CHECK (finalized_at IS NULL OR finalized_at >= created_at)
);

CREATE INDEX idx_referral_attributions_inviter_status
    ON referral_attributions(inviter_user_id, status, id);
