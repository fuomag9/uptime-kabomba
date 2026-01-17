-- Add OAuth support columns to users table
ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN provider TEXT;
ALTER TABLE users ADD COLUMN subject TEXT;
ALTER TABLE users ADD COLUMN oauth_data TEXT;

-- Add unique constraint for email
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email)
WHERE email IS NOT NULL;

-- Add indexes for OAuth lookups
CREATE INDEX IF NOT EXISTS idx_users_provider_subject ON users(provider, subject);

-- Add unique constraint for OAuth users (provider + subject must be unique)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth_unique ON users(provider, subject)
WHERE provider IS NOT NULL AND subject IS NOT NULL;
