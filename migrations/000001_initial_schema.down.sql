-- Drop tables in reverse order
DROP INDEX IF EXISTS idx_api_keys_key;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS api_keys;

DROP TABLE IF EXISTS status_pages;

DROP TABLE IF EXISTS monitor_notifications;

DROP INDEX IF EXISTS idx_notifications_user_id;
DROP TABLE IF EXISTS notifications;

DROP INDEX IF EXISTS idx_stat_daily_monitor_time;
DROP TABLE IF EXISTS stat_daily;

DROP INDEX IF EXISTS idx_stat_hourly_monitor_time;
DROP TABLE IF EXISTS stat_hourly;

DROP INDEX IF EXISTS idx_stat_minutely_monitor_time;
DROP TABLE IF EXISTS stat_minutely;

DROP INDEX IF EXISTS idx_heartbeats_time;
DROP INDEX IF EXISTS idx_heartbeats_monitor_time;
DROP TABLE IF EXISTS heartbeats;

DROP INDEX IF EXISTS idx_monitors_active;
DROP INDEX IF EXISTS idx_monitors_type;
DROP INDEX IF EXISTS idx_monitors_user_id;
DROP TABLE IF EXISTS monitors;

DROP TABLE IF EXISTS users;
