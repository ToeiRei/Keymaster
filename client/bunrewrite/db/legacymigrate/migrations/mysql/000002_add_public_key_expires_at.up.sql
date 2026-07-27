-- Add expires_at column to public_keys (nullable)
ALTER TABLE public_keys
  ADD COLUMN expires_at DATETIME NULL;
