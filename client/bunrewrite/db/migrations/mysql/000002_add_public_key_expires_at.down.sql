-- Remove expires_at column from public_keys
ALTER TABLE public_keys
  DROP COLUMN expires_at;
