-- Rollback drift_events table creation
DROP INDEX idx_drift_remediated ON drift_events;
DROP INDEX idx_drift_type ON drift_events;
DROP INDEX idx_drift_detected ON drift_events;
DROP INDEX idx_drift_account ON drift_events;
DROP TABLE IF EXISTS drift_events;
