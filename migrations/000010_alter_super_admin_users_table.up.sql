ALTER TABLE users ADD COLUMN is_super_admin BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_users_is_super_admin ON users(is_super_admin) WHERE is_super_admin = TRUE;
