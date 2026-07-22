-- Remove expires_at column from public_keys
BEGIN;
ALTER TABLE public_keys DROP COLUMN IF EXISTS expires_at;
COMMIT;
