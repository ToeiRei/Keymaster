-- Add expires_at column to public_keys (nullable)
BEGIN;
ALTER TABLE public_keys ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE;
COMMIT;
