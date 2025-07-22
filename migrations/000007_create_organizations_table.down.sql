DELETE FROM user_organization_access
WHERE organization_id = '00000000-0000-0000-0000-000000000001';

DELETE FROM organizations
WHERE id = '00000000-0000-0000-0000-000000000001';

DROP TABLE IF EXISTS user_organization_access;

DROP TABLE IF EXISTS organizations;