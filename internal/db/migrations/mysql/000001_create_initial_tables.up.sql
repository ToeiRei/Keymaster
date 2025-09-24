-- Copyright (c) 2025 ToeiRei
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- +migrate Up

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    label VARCHAR(255),
    tags TEXT,
    serial INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(username, hostname)
);
CREATE TABLE IF NOT EXISTS public_keys (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    algorithm VARCHAR(255) NOT NULL,
    key_data TEXT NOT NULL,
    comment VARCHAR(255) NOT NULL UNIQUE,
    is_global BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS account_keys (
    account_id INTEGER NOT NULL,
    key_id INTEGER NOT NULL,
    PRIMARY KEY (account_id, key_id),
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
    FOREIGN KEY (key_id) REFERENCES public_keys (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS system_keys (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    serial INTEGER NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS known_hosts (
    hostname VARCHAR(255) NOT NULL PRIMARY KEY,
    `key` TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    details TEXT
);

-- +migrate Down

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS known_hosts;
DROP TABLE IF EXISTS system_keys;
DROP TABLE IF EXISTS account_keys;
DROP TABLE IF EXISTS public_keys;
DROP TABLE IF EXISTS accounts;
