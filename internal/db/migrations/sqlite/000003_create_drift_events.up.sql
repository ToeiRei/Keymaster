-- Create drift_events table for tracking configuration drift
CREATE TABLE drift_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    detected_at TEXT NOT NULL,
    drift_type TEXT NOT NULL CHECK(drift_type IN ('critical', 'warning', 'info')),
    details TEXT,
    was_remediated BOOLEAN NOT NULL DEFAULT 0,
    remediated_at TEXT,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX idx_drift_account ON drift_events(account_id);
CREATE INDEX idx_drift_detected ON drift_events(detected_at);
CREATE INDEX idx_drift_type ON drift_events(drift_type);
CREATE INDEX idx_drift_remediated ON drift_events(was_remediated);
