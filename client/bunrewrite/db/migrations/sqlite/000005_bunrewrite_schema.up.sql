-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Reshape the legacy schema to the bunrewrite model:
--   public_keys.key_data      -> data
--   accounts.hostname         -> host (+ new port/deploy_* columns)
--   account_keys (join table) -> links (direct account<->public_key 1:1 rows)
-- SQLite predates most ALTER operations, so the diverging tables are rebuilt.
-- Foreign key enforcement is off by default for the migration connection, so
-- dropping/recreating parent tables does not disturb the account_keys rows we
-- still read from below.

-- public_keys: rename key_data -> data
CREATE TABLE public_keys_new (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    algorithm TEXT NOT NULL,
    data TEXT NOT NULL,
    comment TEXT NOT NULL UNIQUE,
    is_global BOOLEAN NOT NULL DEFAULT 0,
    expires_at TIMESTAMP
);
INSERT INTO public_keys_new (id, algorithm, data, comment, is_global, expires_at)
    SELECT id, algorithm, key_data, comment, is_global, expires_at FROM public_keys;
DROP TABLE public_keys;
ALTER TABLE public_keys_new RENAME TO public_keys;

-- accounts: hostname -> host, add port/deploy_method/deploy_secret, drop obsolete columns
CREATE TABLE accounts_new (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    host TEXT NOT NULL DEFAULT '',
    port TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    is_dirty BOOLEAN NOT NULL DEFAULT 1,
    deploy_method TEXT NOT NULL DEFAULT '',
    deploy_secret TEXT NOT NULL DEFAULT '',
    UNIQUE(username, host)
);
INSERT INTO accounts_new (id, username, host, port, is_active, is_dirty)
    SELECT id, username, hostname, '', is_active, is_dirty FROM accounts;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;

-- links: direct account <-> public_key relation, backfilled from account_keys
CREATE TABLE IF NOT EXISTS links (
    account_id INTEGER NOT NULL,
    public_key_id INTEGER NOT NULL,
    expires_at TIMESTAMP,
    PRIMARY KEY (account_id, public_key_id),
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
    FOREIGN KEY (public_key_id) REFERENCES public_keys (id) ON DELETE CASCADE
);
INSERT INTO links (account_id, public_key_id, expires_at)
    SELECT account_id, key_id, NULL FROM account_keys;
