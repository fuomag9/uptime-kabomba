-- Drop added indexes
DROP INDEX IF EXISTS idx_incidents_status_page;
DROP INDEX IF EXISTS idx_status_pages_slug;
DROP INDEX IF EXISTS idx_status_pages_user_id;

-- Drop added tables
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS status_page_monitors;

-- Note: To fully revert, you would need to recreate the tables
