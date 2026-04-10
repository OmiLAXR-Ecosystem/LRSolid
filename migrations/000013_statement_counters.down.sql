DROP TRIGGER IF EXISTS trg_increment_statement_counters ON statements;
DROP FUNCTION IF EXISTS increment_statement_counters();

DROP TABLE IF EXISTS statement_stats_daily;
DROP INDEX IF EXISTS idx_statements_client_id;

ALTER TABLE statements
  DROP COLUMN IF EXISTS client_id;

ALTER TABLE clients
  DROP COLUMN IF EXISTS statement_count;

ALTER TABLE lrs
  DROP COLUMN IF EXISTS statement_count;

ALTER TABLE projects
  DROP COLUMN IF EXISTS statement_count;

ALTER TABLE organisations
  DROP COLUMN IF EXISTS statement_count;
