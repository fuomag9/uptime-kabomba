-- Create api_keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL, -- First 8 chars for identification
    scopes TEXT NOT NULL, -- JSON array of scopes: ["read", "write", "admin"]
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix);

-- Create statistics aggregation tables for performance
CREATE TABLE IF NOT EXISTS stat_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    hour TIMESTAMP NOT NULL,
    ping_min INTEGER,
    ping_max INTEGER,
    ping_avg REAL,
    up_count INTEGER DEFAULT 0,
    down_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    uptime_percentage REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(monitor_id, hour),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX idx_stat_hourly_monitor_hour ON stat_hourly(monitor_id, hour DESC);

CREATE TABLE IF NOT EXISTS stat_daily (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    date DATE NOT NULL,
    ping_min INTEGER,
    ping_max INTEGER,
    ping_avg REAL,
    up_count INTEGER DEFAULT 0,
    down_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    uptime_percentage REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(monitor_id, date),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX idx_stat_daily_monitor_date ON stat_daily(monitor_id, date DESC);
