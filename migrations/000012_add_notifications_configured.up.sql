-- Add field to track if notifications have been explicitly configured
ALTER TABLE monitors ADD COLUMN notifications_configured BOOLEAN DEFAULT 0;

-- Mark existing monitors with notification config as configured
UPDATE monitors SET notifications_configured = 1
WHERE id IN (SELECT DISTINCT monitor_id FROM monitor_notifications);
