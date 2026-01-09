-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Add a durable 'is_dirty' flag to accounts. Default FALSE for existing rows.
ALTER TABLE accounts ADD COLUMN is_dirty BOOLEAN NOT NULL DEFAULT 0;
