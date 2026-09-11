CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor TEXT NOT NULL CHECK (length(CAST(actor AS BLOB)) BETWEEN 1 AND 256 AND actor = trim(actor)),
    action TEXT NOT NULL CHECK (length(CAST(action AS BLOB)) BETWEEN 1 AND 256 AND action = trim(action)),
    target TEXT NOT NULL CHECK (length(CAST(target AS BLOB)) BETWEEN 1 AND 256 AND target = trim(target)),
    before_snapshot TEXT CHECK (before_snapshot IS NULL OR length(CAST(before_snapshot AS BLOB)) <= 65536),
    after_snapshot TEXT CHECK (after_snapshot IS NULL OR length(CAST(after_snapshot AS BLOB)) <= 65536),
    request_id TEXT NOT NULL CHECK (length(CAST(request_id AS BLOB)) BETWEEN 1 AND 256 AND request_id = trim(request_id)),
    created_at INTEGER NOT NULL CHECK (created_at > 0)
);

CREATE INDEX idx_audit_log_created_id
    ON audit_log(created_at DESC, id DESC);

CREATE TRIGGER audit_log_no_update
BEFORE UPDATE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;

CREATE TRIGGER audit_log_no_delete
BEFORE DELETE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;
