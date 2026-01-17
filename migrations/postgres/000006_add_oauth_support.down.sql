-- Remove OAuth support from users table
DROP INDEX IF EXISTS idx_users_oauth_unique;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_provider_subject;

-- Note: Consider creating a new table and migrating data
-- ALTER TABLE users DROP COLUMN oauth_data;
-- ALTER TABLE users DROP COLUMN subject;
-- ALTER TABLE users DROP COLUMN provider;
-- ALTER TABLE users DROP COLUMN email;
