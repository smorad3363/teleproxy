ALTER TABLE referral_attributions
    ADD COLUMN eligible_at INTEGER
    CHECK (
        eligible_at IS NULL OR (
            eligible_at >= created_at
            AND status IN ('pending', 'rewarded')
            AND rejection_reason IS NULL
            AND (status = 'rewarded' OR finalized_at IS NULL)
        )
    );

CREATE INDEX idx_referral_attributions_pending_eligibility
    ON referral_attributions(status, eligible_at, id)
    WHERE status = 'pending';
