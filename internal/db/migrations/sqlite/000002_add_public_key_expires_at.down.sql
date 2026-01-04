-- Remove expires_at column from public_keys
-- SQLite doesn't support DROP COLUMN; recreate table without expires_at
CREATE TABLE public_keys_new (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    algorithm TEXT NOT NULL,
    key_data TEXT NOT NULL,
    comment TEXT NOT NULL UNIQUE,
    is_global BOOLEAN NOT NULL DEFAULT 0
);
INSERT INTO public_keys_new (id, algorithm, key_data, comment, is_global)
  SELECT id, algorithm, key_data, comment, is_global FROM public_keys;
DROP TABLE public_keys;
ALTER TABLE public_keys_new RENAME TO public_keys;
