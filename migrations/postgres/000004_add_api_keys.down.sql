-- Drop statistics tables
DROP INDEX IF EXISTS idx_stat_daily_monitor_date;
DROP TABLE IF EXISTS stat_daily;

DROP INDEX IF EXISTS idx_stat_hourly_monitor_hour;
DROP TABLE IF EXISTS stat_hourly;

-- Drop api_keys table
DROP INDEX IF EXISTS idx_api_keys_prefix;
DROP INDEX IF EXISTS idx_api_keys_key_hash;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS api_keys;
