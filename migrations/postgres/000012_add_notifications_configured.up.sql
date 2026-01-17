-- Add field to track if notifications have been explicitly configured
ALTER TABLE monitors ADD COLUMN notifications_configured BOOLEAN DEFAULT false;

-- Mark existing monitors with notification config as configured
UPDATE monitors SET notifications_configured = true
WHERE id IN (SELECT DISTINCT monitor_id FROM monitor_notifications);
