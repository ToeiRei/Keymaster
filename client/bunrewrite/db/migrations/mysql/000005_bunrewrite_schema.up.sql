-- Copyright (c) 2026 Keymaster Team
-- Keymaster - SSH key management system
-- This source code is licensed under the MIT license found in the LICENSE file.

-- Reshape the legacy schema to the bunrewrite model using in-place ALTERs.

-- public_keys: rename key_data -> data
ALTER TABLE public_keys RENAME COLUMN key_data TO data;

-- accounts: hostname -> host, add port/deploy_method/deploy_secret, drop obsolete columns
ALTER TABLE accounts RENAME COLUMN hostname TO host;
ALTER TABLE accounts ADD COLUMN port VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN deploy_method VARCHAR(255) NOT NULL DEFAULT '';
-- TEXT columns cannot take a literal DEFAULT on older MySQL; add nullable then backfill.
ALTER TABLE accounts ADD COLUMN deploy_secret TEXT;
UPDATE accounts SET deploy_secret = '' WHERE deploy_secret IS NULL;
ALTER TABLE accounts DROP COLUMN label;
ALTER TABLE accounts DROP COLUMN tags;
ALTER TABLE accounts DROP COLUMN serial;
ALTER TABLE accounts DROP COLUMN key_hash;

-- links: direct account <-> public_key relation, backfilled from account_keys
CREATE TABLE IF NOT EXISTS links (
    account_id INTEGER NOT NULL,
    public_key_id INTEGER NOT NULL,
    expires_at TIMESTAMP NULL,
    PRIMARY KEY (account_id, public_key_id),
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
    FOREIGN KEY (public_key_id) REFERENCES public_keys (id) ON DELETE CASCADE
);
INSERT INTO links (account_id, public_key_id, expires_at)
    SELECT account_id, key_id, NULL FROM account_keys;
