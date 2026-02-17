-- Add schedule_interval_hours to brands table
ALTER TABLE brands ADD COLUMN IF NOT EXISTS schedule_interval_hours INTEGER DEFAULT 4;
