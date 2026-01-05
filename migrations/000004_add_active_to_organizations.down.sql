-- Remove `active` column from organizations (down migration)
ALTER TABLE organizations
    DROP COLUMN IF EXISTS active;
