-- Revert api_keys table to old structure
CREATE TABLE IF NOT EXISTS api_keys_old (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    key TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data back (key_hash becomes key)
INSERT INTO api_keys_old (id, user_id, name, key, expires_at, created_at)
SELECT id, user_id, name, key_hash, expires_at, created_at
FROM api_keys;

-- Drop new table
DROP TABLE api_keys;

-- Rename old table back
ALTER TABLE api_keys_old RENAME TO api_keys;

-- Create old indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
