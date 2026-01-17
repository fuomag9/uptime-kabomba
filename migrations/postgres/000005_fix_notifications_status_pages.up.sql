-- Note: All columns already exist in initial schema (000001_initial_schema.up.sql)
-- This migration only creates the mapping tables and additional indexes

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
    id SERIAL PRIMARY KEY,
    status_page_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    style TEXT DEFAULT 'info',
    pin BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE
);

-- Add additional indexes (idx_status_pages_user_id and idx_status_pages_slug already exist in 000001)
CREATE INDEX IF NOT EXISTS idx_incidents_status_page ON incidents(status_page_id);
