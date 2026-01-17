-- Modernize api_keys table to match current model
-- Recreate the table to update schema safely

-- Create new table with updated structure
CREATE TABLE IF NOT EXISTS api_keys_new (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    scopes TEXT,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy existing data (set prefix to empty string for old keys, they'll need to be regenerated)
INSERT INTO api_keys_new (id, user_id, name, key_hash, prefix, expires_at, created_at)
SELECT id, user_id, name, key as key_hash, '', expires_at, created_at
FROM api_keys;

-- Ensure the sequence is set to the current max id
SELECT setval(
    pg_get_serial_sequence('api_keys_new', 'id'),
    GREATEST(1, COALESCE(MAX(id), 0))
)
FROM api_keys_new;

-- Drop old table
DROP TABLE api_keys;

-- Rename new table
ALTER TABLE api_keys_new RENAME TO api_keys;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);
