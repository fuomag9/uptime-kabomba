-- Refresh tokens: opaque, high-entropy random tokens stored as a fast hash
-- (not bcrypt - these tokens are 256 bits of random entropy, so a slow,
-- salted KDF buys nothing over SHA-256 and would prevent an indexed lookup).
-- Rotated on every use; revoked_at is set instead of deleting so a reused
-- (already-rotated) token can be detected as stolen.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Revoked access (JWT) tokens, keyed by jti. JWTs are normally stateless and
-- can't be individually invalidated before they expire; this table is the
-- exception list checked on every authenticated request so that logout (or
-- an admin-initiated revocation) can actually kill a live access token
-- instead of only clearing the cookie client-side.
CREATE TABLE IF NOT EXISTS revoked_access_tokens (
    jti TEXT PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_revoked_access_tokens_expires_at ON revoked_access_tokens(expires_at);
