-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

ALTER TABLE accounts DROP COLUMN IF EXISTS is_dirty;
