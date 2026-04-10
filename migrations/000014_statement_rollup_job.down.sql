DROP INDEX IF EXISTS idx_statements_rollup_cursor;
DROP TABLE IF EXISTS statement_rollup_state;

-- Restore trigger-based counter updates.
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

  UPDATE lrs SET statement_count = statement_count + 1 WHERE id = NEW.tenant_id;
  IF v_project_id IS NOT NULL THEN
    UPDATE projects SET statement_count = statement_count + 1 WHERE id = v_project_id;
  END IF;
  IF v_org_id IS NOT NULL THEN
    UPDATE organisations SET statement_count = statement_count + 1 WHERE id = v_org_id;
  END IF;
  IF NEW.client_id IS NOT NULL THEN
    UPDATE clients SET statement_count = statement_count + 1 WHERE id = NEW.client_id;
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
