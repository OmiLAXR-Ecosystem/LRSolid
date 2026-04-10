ALTER TABLE organisations
  ADD COLUMN IF NOT EXISTS statement_count bigint NOT NULL DEFAULT 0;

ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS statement_count bigint NOT NULL DEFAULT 0;

ALTER TABLE lrs
  ADD COLUMN IF NOT EXISTS statement_count bigint NOT NULL DEFAULT 0;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS statement_count bigint NOT NULL DEFAULT 0;

ALTER TABLE statements
  ADD COLUMN IF NOT EXISTS client_id uuid NULL REFERENCES clients(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_statements_client_id ON statements(client_id, stored_at DESC);

CREATE TABLE IF NOT EXISTS statement_stats_daily (
  day             date        NOT NULL,
  scope_type      text        NOT NULL,
  scope_id        text        NOT NULL,
  statement_count bigint      NOT NULL DEFAULT 0,
  PRIMARY KEY (day, scope_type, scope_id),
  CONSTRAINT chk_statement_stats_daily_scope_type
    CHECK (scope_type IN ('global', 'organisation', 'project', 'lrs', 'client'))
);

CREATE OR REPLACE FUNCTION increment_statement_counters() RETURNS trigger AS $$
DECLARE
  v_day date;
  v_project_id uuid;
  v_org_id uuid;
BEGIN
  v_day := (NEW.stored_at AT TIME ZONE 'UTC')::date;

  SELECT l.project_id, p.org_id
    INTO v_project_id, v_org_id
  FROM lrs l
  JOIN projects p ON p.id = l.project_id
  WHERE l.id = NEW.tenant_id;

  UPDATE lrs
     SET statement_count = statement_count + 1
   WHERE id = NEW.tenant_id;

  IF v_project_id IS NOT NULL THEN
    UPDATE projects
       SET statement_count = statement_count + 1
     WHERE id = v_project_id;
  END IF;

  IF v_org_id IS NOT NULL THEN
    UPDATE organisations
       SET statement_count = statement_count + 1
     WHERE id = v_org_id;
  END IF;

  IF NEW.client_id IS NOT NULL THEN
    UPDATE clients
       SET statement_count = statement_count + 1
     WHERE id = NEW.client_id;
  END IF;

  INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
  VALUES (v_day, 'global', '*', 1)
  ON CONFLICT (day, scope_type, scope_id)
  DO UPDATE SET statement_count = statement_stats_daily.statement_count + 1;

  IF v_org_id IS NOT NULL THEN
    INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
    VALUES (v_day, 'organisation', v_org_id::text, 1)
    ON CONFLICT (day, scope_type, scope_id)
    DO UPDATE SET statement_count = statement_stats_daily.statement_count + 1;
  END IF;

  IF v_project_id IS NOT NULL THEN
    INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
    VALUES (v_day, 'project', v_project_id::text, 1)
    ON CONFLICT (day, scope_type, scope_id)
    DO UPDATE SET statement_count = statement_stats_daily.statement_count + 1;
  END IF;

  INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
  VALUES (v_day, 'lrs', NEW.tenant_id, 1)
  ON CONFLICT (day, scope_type, scope_id)
  DO UPDATE SET statement_count = statement_stats_daily.statement_count + 1;

  IF NEW.client_id IS NOT NULL THEN
    INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
    VALUES (v_day, 'client', NEW.client_id::text, 1)
    ON CONFLICT (day, scope_type, scope_id)
    DO UPDATE SET statement_count = statement_stats_daily.statement_count + 1;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_increment_statement_counters ON statements;
CREATE TRIGGER trg_increment_statement_counters
AFTER INSERT ON statements
FOR EACH ROW
EXECUTE FUNCTION increment_statement_counters();

-- Backfill counters for existing data.
UPDATE organisations SET statement_count = 0;
UPDATE projects SET statement_count = 0;
UPDATE lrs SET statement_count = 0;
UPDATE clients SET statement_count = 0;

UPDATE lrs l
SET statement_count = s.cnt
FROM (
  SELECT tenant_id, COUNT(*)::bigint AS cnt
  FROM statements
  GROUP BY tenant_id
) s
WHERE l.id = s.tenant_id;

UPDATE projects p
SET statement_count = s.cnt
FROM (
  SELECT l.project_id, COUNT(*)::bigint AS cnt
  FROM statements st
  JOIN lrs l ON l.id = st.tenant_id
  GROUP BY l.project_id
) s
WHERE p.id = s.project_id;

UPDATE organisations o
SET statement_count = s.cnt
FROM (
  SELECT p.org_id, COUNT(*)::bigint AS cnt
  FROM statements st
  JOIN lrs l ON l.id = st.tenant_id
  JOIN projects p ON p.id = l.project_id
  GROUP BY p.org_id
) s
WHERE o.id = s.org_id;

UPDATE clients c
SET statement_count = s.cnt
FROM (
  SELECT client_id, COUNT(*)::bigint AS cnt
  FROM statements
  WHERE client_id IS NOT NULL
  GROUP BY client_id
) s
WHERE c.id = s.client_id;

TRUNCATE TABLE statement_stats_daily;

INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
SELECT (stored_at AT TIME ZONE 'UTC')::date, 'global', '*', COUNT(*)::bigint
FROM statements
GROUP BY 1;

INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
SELECT (st.stored_at AT TIME ZONE 'UTC')::date, 'organisation', p.org_id::text, COUNT(*)::bigint
FROM statements st
JOIN lrs l ON l.id = st.tenant_id
JOIN projects p ON p.id = l.project_id
GROUP BY 1, 2, 3
ON CONFLICT (day, scope_type, scope_id)
DO UPDATE SET statement_count = EXCLUDED.statement_count;

INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
SELECT (st.stored_at AT TIME ZONE 'UTC')::date, 'project', l.project_id::text, COUNT(*)::bigint
FROM statements st
JOIN lrs l ON l.id = st.tenant_id
GROUP BY 1, 2, 3
ON CONFLICT (day, scope_type, scope_id)
DO UPDATE SET statement_count = EXCLUDED.statement_count;

INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
SELECT (stored_at AT TIME ZONE 'UTC')::date, 'lrs', tenant_id, COUNT(*)::bigint
FROM statements
GROUP BY 1, 2, 3
ON CONFLICT (day, scope_type, scope_id)
DO UPDATE SET statement_count = EXCLUDED.statement_count;

INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
SELECT (stored_at AT TIME ZONE 'UTC')::date, 'client', client_id::text, COUNT(*)::bigint
FROM statements
WHERE client_id IS NOT NULL
GROUP BY 1, 2, 3
ON CONFLICT (day, scope_type, scope_id)
DO UPDATE SET statement_count = EXCLUDED.statement_count;
