-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Best-effort reverse: drop the links table introduced by the up migration.
-- The accounts/public_keys column reshaping is not automatically reverted.
DROP TABLE IF EXISTS links;
