-- Add resend_interval column to monitors table
-- This controls how often to resend down notifications
-- Default: 0 = send notification only once per downtime period (never resend)
-- Value of 1 = send notification on every consecutive failure
-- Value of 3 = send on 3rd consecutive failure, then every 3 failures after
ALTER TABLE monitors ADD COLUMN resend_interval INTEGER DEFAULT 0;
