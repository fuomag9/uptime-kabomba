-- User settings for configurable retention periods
CREATE TABLE IF NOT EXISTS user_settings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    heartbeat_retention_days INTEGER DEFAULT 90 NOT NULL,
    hourly_stat_retention_days INTEGER DEFAULT 365 NOT NULL,
    daily_stat_retention_days INTEGER DEFAULT 730 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (heartbeat_retention_days >= 7 AND heartbeat_retention_days <= 365),
    CHECK (hourly_stat_retention_days >= 30 AND hourly_stat_retention_days <= 730),
    CHECK (daily_stat_retention_days >= 90 AND daily_stat_retention_days <= 1825)
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);
