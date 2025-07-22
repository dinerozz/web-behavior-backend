DROP INDEX IF EXISTS idx_extension_users_organization;
ALTER TABLE extension_users DROP COLUMN IF EXISTS organization_id;
