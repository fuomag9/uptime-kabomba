-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT NOT NULL, -- JSON configuration
    is_default BOOLEAN DEFAULT 0,
    active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_active ON notifications(active);

-- Create monitor_notifications junction table
CREATE TABLE IF NOT EXISTS monitor_notifications (
    monitor_id INTEGER NOT NULL,
    notification_id INTEGER NOT NULL,
    PRIMARY KEY (monitor_id, notification_id),
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE INDEX idx_monitor_notifications_monitor ON monitor_notifications(monitor_id);
CREATE INDEX idx_monitor_notifications_notification ON monitor_notifications(notification_id);
