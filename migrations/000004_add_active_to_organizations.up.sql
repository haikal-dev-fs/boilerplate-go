-- Add `active` column to organizations if missing
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;

-- Set existing rows to active = TRUE (in case some are NULL)
UPDATE organizations SET active = TRUE WHERE active IS NULL;
