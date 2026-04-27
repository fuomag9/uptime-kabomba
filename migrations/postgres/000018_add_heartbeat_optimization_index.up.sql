-- Add optimized index for latest heartbeat queries (DISTINCT ON monitor_id ORDER BY monitor_id, time DESC, id DESC)
-- This composite index covers both the DISTINCT ON sort and the WHERE monitor_id IN (...) filter
-- Note: CONCURRENTLY is not used here because golang-migrate runs migrations inside transactions,
-- and CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
CREATE INDEX IF NOT EXISTS idx_heartbeats_monitor_time_id_desc
ON heartbeats(monitor_id, time DESC, id DESC);