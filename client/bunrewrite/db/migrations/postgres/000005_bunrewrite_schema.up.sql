-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Reshape the legacy schema to the bunrewrite model using in-place ALTERs.

-- public_keys: rename key_data -> data
ALTER TABLE public_keys RENAME COLUMN key_data TO data;

-- accounts: hostname -> host, add port/deploy_method/deploy_secret, drop obsolete columns
ALTER TABLE accounts RENAME COLUMN hostname TO host;
ALTER TABLE accounts ADD COLUMN port TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN deploy_method TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN deploy_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts DROP COLUMN IF EXISTS label;
ALTER TABLE accounts DROP COLUMN IF EXISTS tags;
ALTER TABLE accounts DROP COLUMN IF EXISTS serial;
ALTER TABLE accounts DROP COLUMN IF EXISTS key_hash;

-- links: direct account <-> public_key relation, backfilled from account_keys
CREATE TABLE IF NOT EXISTS links (
    account_id INTEGER NOT NULL,
    public_key_id INTEGER NOT NULL,
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (account_id, public_key_id),
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
    FOREIGN KEY (public_key_id) REFERENCES public_keys (id) ON DELETE CASCADE
);
INSERT INTO links (account_id, public_key_id, expires_at)
    SELECT account_id, key_id, NULL FROM account_keys;
