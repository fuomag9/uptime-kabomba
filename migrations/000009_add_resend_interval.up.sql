-- Add resend_interval column to monitors table
-- This controls how often to resend down notifications
-- Default: 1 = send notification on first failure and every subsequent failure
-- Value of 3 = send on 3rd consecutive failure, then every failure after
ALTER TABLE monitors ADD COLUMN resend_interval INTEGER DEFAULT 1;
