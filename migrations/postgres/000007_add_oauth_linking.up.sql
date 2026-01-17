-- Create table for temporary OAuth account linking tokens
CREATE TABLE IF NOT EXISTS oauth_linking_tokens (
    id SERIAL PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    email TEXT NOT NULL,
    oauth_data TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_oauth_linking_tokens_token ON oauth_linking_tokens(token);
CREATE INDEX IF NOT EXISTS idx_oauth_linking_tokens_expires ON oauth_linking_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_linking_tokens_user ON oauth_linking_tokens(user_id);
