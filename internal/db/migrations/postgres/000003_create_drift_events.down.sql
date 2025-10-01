-- Rollback drift_events table creation
DROP INDEX IF EXISTS idx_drift_remediated;
DROP INDEX IF EXISTS idx_drift_type;
DROP INDEX IF EXISTS idx_drift_detected;
DROP INDEX IF EXISTS idx_drift_account;
DROP TABLE IF EXISTS drift_events;
