-- Drop incidents table
DROP INDEX IF EXISTS idx_incidents_created_at;
DROP INDEX IF EXISTS idx_incidents_pin;
DROP INDEX IF EXISTS idx_incidents_status_page;
DROP TABLE IF EXISTS incidents;

-- Drop status_page_monitors junction table
DROP INDEX IF EXISTS idx_status_page_monitors_monitor;
DROP INDEX IF EXISTS idx_status_page_monitors_page;
DROP TABLE IF EXISTS status_page_monitors;

-- Drop status_pages table
DROP INDEX IF EXISTS idx_status_pages_published;
DROP INDEX IF EXISTS idx_status_pages_user_id;
DROP INDEX IF EXISTS idx_status_pages_slug;
DROP TABLE IF EXISTS status_pages;
