-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Add a per-account `key_hash` to store a deterministic fingerprint of
-- the effective authorized_keys content for the account. NULL for existing rows.
ALTER TABLE accounts ADD COLUMN key_hash TEXT;
