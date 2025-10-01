-- Create drift_events table for tracking configuration drift
CREATE TABLE drift_events (
    id INT AUTO_INCREMENT PRIMARY KEY,
    account_id INT NOT NULL,
    detected_at TIMESTAMP NOT NULL,
    drift_type VARCHAR(20) NOT NULL CHECK(drift_type IN ('critical', 'warning', 'info')),
    details TEXT,
    was_remediated BOOLEAN NOT NULL DEFAULT FALSE,
    remediated_at TIMESTAMP NULL,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX idx_drift_account ON drift_events(account_id);
CREATE INDEX idx_drift_detected ON drift_events(detected_at);
CREATE INDEX idx_drift_type ON drift_events(drift_type);
CREATE INDEX idx_drift_remediated ON drift_events(was_remediated);
