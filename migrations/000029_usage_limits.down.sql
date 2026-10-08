DROP TABLE IF EXISTS usage_limits;
DROP TABLE IF EXISTS statement_stats_hourly;

ALTER TABLE organisations DROP COLUMN IF EXISTS storage_bytes;
ALTER TABLE projects      DROP COLUMN IF EXISTS storage_bytes;
ALTER TABLE lrs           DROP COLUMN IF EXISTS storage_bytes;
ALTER TABLE clients       DROP COLUMN IF EXISTS storage_bytes;
