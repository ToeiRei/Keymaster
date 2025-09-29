-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

DROP INDEX IF EXISTS idx_bootstrap_sessions_status;
DROP INDEX IF EXISTS idx_bootstrap_sessions_expires_at;
DROP TABLE IF EXISTS bootstrap_sessions;