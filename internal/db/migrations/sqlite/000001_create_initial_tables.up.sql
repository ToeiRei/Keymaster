-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    hostname TEXT NOT NULL,
    label TEXT,
    tags TEXT,
    serial INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    UNIQUE(username, hostname)
);
CREATE TABLE IF NOT EXISTS public_keys (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    algorithm TEXT NOT NULL,
    key_data TEXT NOT NULL,
    comment TEXT NOT NULL UNIQUE,
    is_global BOOLEAN NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS account_keys (
    account_id INTEGER NOT NULL,
    key_id INTEGER NOT NULL,
    PRIMARY KEY (account_id, key_id),
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
    FOREIGN KEY (key_id) REFERENCES public_keys (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS system_keys (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    serial INTEGER NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS known_hosts (
    hostname TEXT NOT NULL PRIMARY KEY,
    key TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT
);
