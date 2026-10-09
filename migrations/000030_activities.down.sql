DROP TABLE IF EXISTS activities;

DROP INDEX IF EXISTS idx_statements_activity_ids;

ALTER TABLE statements DROP COLUMN IF EXISTS activity_ids;
