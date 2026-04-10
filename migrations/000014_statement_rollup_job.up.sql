-- Switch from per-row trigger increments to async rollup job.
DROP TRIGGER IF EXISTS trg_increment_statement_counters ON statements;
DROP FUNCTION IF EXISTS increment_statement_counters();

CREATE TABLE IF NOT EXISTS statement_rollup_state (
  id                integer     PRIMARY KEY,
  last_stored_at    timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00',
  last_tenant_id    text        NOT NULL DEFAULT '',
  last_statement_id text        NOT NULL DEFAULT '',
  updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_statements_rollup_cursor
  ON statements(stored_at, tenant_id, id);

INSERT INTO statement_rollup_state (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

-- Set watermark to current max to avoid double-counting historical data
-- (000013 already backfilled counters).
WITH mx AS (
  SELECT stored_at, tenant_id, id
  FROM statements
  ORDER BY stored_at DESC, tenant_id DESC, id DESC
  LIMIT 1
)
UPDATE statement_rollup_state s
SET
  last_stored_at = COALESCE((SELECT stored_at FROM mx), '1970-01-01 00:00:00+00'),
  last_tenant_id = COALESCE((SELECT tenant_id FROM mx), ''),
  last_statement_id = COALESCE((SELECT id FROM mx), ''),
  updated_at = now()
WHERE s.id = 1;
