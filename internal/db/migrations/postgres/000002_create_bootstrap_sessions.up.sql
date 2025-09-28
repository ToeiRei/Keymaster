-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

CREATE TABLE IF NOT EXISTS bootstrap_sessions (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    hostname TEXT NOT NULL,
    label TEXT,
    tags TEXT,
    temp_public_key TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
);

-- Create index for efficient cleanup of expired sessions
CREATE INDEX IF NOT EXISTS idx_bootstrap_sessions_expires_at ON bootstrap_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_bootstrap_sessions_status ON bootstrap_sessions(status);