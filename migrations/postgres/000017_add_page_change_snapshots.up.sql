CREATE TABLE page_change_snapshots (
    id              SERIAL PRIMARY KEY,
    monitor_id      INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    heartbeat_id    INTEGER REFERENCES heartbeats(id) ON DELETE SET NULL,
    screenshot_path TEXT,
    baseline_path   TEXT,
    diff_path       TEXT,
    html_hash       TEXT,
    runtime_metrics JSONB,
    change_score    REAL NOT NULL DEFAULT 0,
    image_score     REAL NOT NULL DEFAULT 0,
    html_score      REAL NOT NULL DEFAULT 0,
    runtime_score   REAL NOT NULL DEFAULT 0,
    is_baseline     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pcs_monitor_id ON page_change_snapshots(monitor_id);
CREATE INDEX idx_pcs_baseline ON page_change_snapshots(monitor_id, is_baseline) WHERE is_baseline = TRUE;
