-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Remove the `key_hash` column by recreating the table without it.
PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
CREATE TABLE accounts_new (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    hostname TEXT NOT NULL,
    label TEXT,
    tags TEXT,
    serial INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    is_dirty BOOLEAN NOT NULL DEFAULT 0,
    UNIQUE(username, hostname)
);
INSERT INTO accounts_new (id, username, hostname, label, tags, serial, is_active, is_dirty)
    SELECT id, username, hostname, label, tags, serial, is_active, is_dirty FROM accounts;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;
COMMIT;
PRAGMA foreign_keys=on;
