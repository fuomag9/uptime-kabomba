-- Create status_pages table
CREATE TABLE IF NOT EXISTS status_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    published BOOLEAN DEFAULT 0,
    show_powered_by BOOLEAN DEFAULT 1,
    theme TEXT DEFAULT 'light',
    custom_css TEXT,
    password TEXT, -- bcrypt hash for password protection
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_status_pages_slug ON status_pages(slug);
CREATE INDEX idx_status_pages_user_id ON status_pages(user_id);
CREATE INDEX idx_status_pages_published ON status_pages(published);

-- Create status_page_monitors junction table
CREATE TABLE IF NOT EXISTS status_page_monitors (
    status_page_id INTEGER NOT NULL,
    monitor_id INTEGER NOT NULL,
    display_order INTEGER DEFAULT 0,
    PRIMARY KEY (status_page_id, monitor_id),
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

CREATE INDEX idx_status_page_monitors_page ON status_page_monitors(status_page_id);
CREATE INDEX idx_status_page_monitors_monitor ON status_page_monitors(monitor_id);

-- Create incidents table for status page incidents
CREATE TABLE IF NOT EXISTS incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status_page_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    style TEXT DEFAULT 'info', -- info, warning, danger, success
    pin BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE
);

CREATE INDEX idx_incidents_status_page ON incidents(status_page_id);
CREATE INDEX idx_incidents_pin ON incidents(pin);
CREATE INDEX idx_incidents_created_at ON incidents(created_at DESC);
