-- Add missing columns to notifications table
ALTER TABLE notifications ADD COLUMN active BOOLEAN DEFAULT 1;
ALTER TABLE notifications ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Add missing columns to status_pages table
ALTER TABLE status_pages ADD COLUMN user_id INTEGER;
ALTER TABLE status_pages ADD COLUMN show_powered_by BOOLEAN DEFAULT 1;
ALTER TABLE status_pages ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create status_page_monitors mapping table
CREATE TABLE IF NOT EXISTS status_page_monitors (
    status_page_id INTEGER NOT NULL,
    monitor_id INTEGER NOT NULL,
    PRIMARY KEY (status_page_id, monitor_id),
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

-- Create incidents table
CREATE TABLE IF NOT EXISTS incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status_page_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    style TEXT DEFAULT 'info',
    pin BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_status_pages_user_id ON status_pages(user_id);
CREATE INDEX IF NOT EXISTS idx_status_pages_slug ON status_pages(slug);
CREATE INDEX IF NOT EXISTS idx_incidents_status_page ON incidents(status_page_id);
