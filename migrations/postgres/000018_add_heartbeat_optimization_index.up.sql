-- Add optimized index for latest heartbeat queries
-- This index is optimized for MAX(time) GROUP BY monitor_id queries
-- which are used to fetch the latest heartbeat per monitor
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_heartbeats_monitor_max_time 
ON heartbeats(monitor_id, time);
