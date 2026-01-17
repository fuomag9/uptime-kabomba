-- Drop monitor_notifications junction table
DROP INDEX IF EXISTS idx_monitor_notifications_notification;
DROP INDEX IF EXISTS idx_monitor_notifications_monitor;
DROP TABLE IF EXISTS monitor_notifications;

-- Drop notifications table
DROP INDEX IF EXISTS idx_notifications_active;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP TABLE IF EXISTS notifications;
