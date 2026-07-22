-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

CREATE TABLE IF NOT EXISTS bootstrap_sessions (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    label TEXT,
    tags TEXT,
    temp_public_key TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active'
);

-- Create index for efficient cleanup of expired sessions
CREATE INDEX idx_bootstrap_sessions_expires_at ON bootstrap_sessions(expires_at);
CREATE INDEX idx_bootstrap_sessions_status ON bootstrap_sessions(status);