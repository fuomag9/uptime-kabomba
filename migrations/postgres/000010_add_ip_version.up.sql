-- Add ip_version column to monitors table
-- Allows forcing IPv4 or IPv6 for network checks
-- Values: 'auto', 'ipv4', 'ipv6'
ALTER TABLE monitors ADD COLUMN ip_version TEXT DEFAULT 'auto';
