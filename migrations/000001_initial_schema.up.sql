-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    active BOOLEAN DEFAULT 1,
    totp_secret TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Monitors table
CREATE TABLE IF NOT EXISTS monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    url TEXT,
    interval INTEGER DEFAULT 60,
    timeout INTEGER DEFAULT 30,
    active BOOLEAN DEFAULT 1,
    config TEXT, -- JSON blob for type-specific settings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_monitors_user_id ON monitors(user_id);
CREATE INDEX IF NOT EXISTS idx_monitors_type ON monitors(type);
CREATE INDEX IF NOT EXISTS idx_monitors_active ON monitors(active);

-- Heartbeats table
CREATE TABLE IF NOT EXISTS heartbeats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    status INTEGER NOT NULL, -- 0=down, 1=up, 2=pending, 3=maintenance
    ping INTEGER, -- milliseconds
    important BOOLEAN DEFAULT 0,
    message TEXT,
    time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_heartbeats_monitor_time ON heartbeats(monitor_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON heartbeats(time);

-- Statistics tables
CREATE TABLE IF NOT EXISTS stat_minutely (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    ping_min INTEGER,
    ping_max INTEGER,
    ping_avg REAL,
    up_count INTEGER,
    down_count INTEGER,
    uptime_percentage REAL,
    UNIQUE(monitor_id, timestamp),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stat_minutely_monitor_time ON stat_minutely(monitor_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS stat_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    ping_min INTEGER,
    ping_max INTEGER,
    ping_avg REAL,
    up_count INTEGER,
    down_count INTEGER,
    uptime_percentage REAL,
    UNIQUE(monitor_id, timestamp),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stat_hourly_monitor_time ON stat_hourly(monitor_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS stat_daily (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    ping_min INTEGER,
    ping_max INTEGER,
    ping_avg REAL,
    up_count INTEGER,
    down_count INTEGER,
    uptime_percentage REAL,
    UNIQUE(monitor_id, timestamp),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stat_daily_monitor_time ON stat_daily(monitor_id, timestamp DESC);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT NOT NULL, -- JSON blob
    is_default BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);

-- Monitor-Notification mapping
CREATE TABLE IF NOT EXISTS monitor_notifications (
    monitor_id INTEGER NOT NULL,
    notification_id INTEGER NOT NULL,
    PRIMARY KEY (monitor_id, notification_id),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

-- Status pages table
CREATE TABLE IF NOT EXISTS status_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    published BOOLEAN DEFAULT 0,
    theme TEXT DEFAULT 'light',
    custom_css TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    key TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
