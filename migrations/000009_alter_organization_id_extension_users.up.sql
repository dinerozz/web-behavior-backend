ALTER TABLE extension_users
    ADD COLUMN organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE;

CREATE INDEX idx_extension_users_organization ON extension_users(organization_id);

 select * from organizations;