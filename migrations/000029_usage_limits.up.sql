-- Quotas per organisation, project, LRS, or client. A rule caps the
-- aggregate already stored on that row (everything beneath it).

CREATE TABLE IF NOT EXISTS usage_limits (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_type  text        NOT NULL,
  subject_id    text        NOT NULL,
  metric        text        NOT NULL,
  max_value     bigint      NOT NULL,
  enabled       boolean     NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_usage_limits_subject_type
    CHECK (subject_type IN ('organisation', 'project', 'lrs', 'client')),
  CONSTRAINT chk_usage_limits_metric
    CHECK (metric IN ('statements_total', 'statements_per_hour', 'statements_per_day', 'storage_bytes')),
  CONSTRAINT chk_usage_limits_max_value CHECK (max_value >= 0),
  UNIQUE (subject_type, subject_id, metric)
);

CREATE TABLE IF NOT EXISTS statement_stats_hourly (
  hour            timestamptz NOT NULL,
  scope_type      text        NOT NULL,
  scope_id        text        NOT NULL,
  statement_count bigint      NOT NULL DEFAULT 0,
  PRIMARY KEY (hour, scope_type, scope_id),
  CONSTRAINT chk_statement_stats_hourly_scope_type
    CHECK (scope_type IN ('global', 'organisation', 'project', 'lrs', 'client'))
);

ALTER TABLE organisations ADD COLUMN IF NOT EXISTS storage_bytes bigint NOT NULL DEFAULT 0;
ALTER TABLE projects      ADD COLUMN IF NOT EXISTS storage_bytes bigint NOT NULL DEFAULT 0;
ALTER TABLE lrs           ADD COLUMN IF NOT EXISTS storage_bytes bigint NOT NULL DEFAULT 0;
ALTER TABLE clients       ADD COLUMN IF NOT EXISTS storage_bytes bigint NOT NULL DEFAULT 0;

-- Count only rows the rollup worker has already applied, so the next batch
-- does not add them again. Payload size is the JSON text, same unit the
-- ingest path charges for external stores.
UPDATE lrs l
SET storage_bytes = COALESCE((
  SELECT SUM(octet_length(st.statement::text))::bigint
  FROM statements st
  JOIN statement_rollup_state wm ON wm.id = 1
  WHERE st.tenant_id = l.id
    AND (st.stored_at, st.tenant_id, st.id) <= (wm.last_stored_at, wm.last_tenant_id, wm.last_statement_id)
), 0);

UPDATE projects p
SET storage_bytes = COALESCE((
  SELECT SUM(octet_length(st.statement::text))::bigint
  FROM statements st
  JOIN lrs l ON l.id = st.tenant_id
  JOIN statement_rollup_state wm ON wm.id = 1
  WHERE l.project_id = p.id
    AND (st.stored_at, st.tenant_id, st.id) <= (wm.last_stored_at, wm.last_tenant_id, wm.last_statement_id)
), 0);

UPDATE organisations o
SET storage_bytes = COALESCE((
  SELECT SUM(octet_length(st.statement::text))::bigint
  FROM statements st
  JOIN lrs l ON l.id = st.tenant_id
  JOIN projects p ON p.id = l.project_id
  JOIN statement_rollup_state wm ON wm.id = 1
  WHERE p.org_id = o.id
    AND (st.stored_at, st.tenant_id, st.id) <= (wm.last_stored_at, wm.last_tenant_id, wm.last_statement_id)
), 0);

UPDATE clients c
SET storage_bytes = COALESCE((
  SELECT SUM(octet_length(st.statement::text))::bigint
  FROM statements st
  JOIN statement_rollup_state wm ON wm.id = 1
  WHERE st.client_id = c.id
    AND (st.stored_at, st.tenant_id, st.id) <= (wm.last_stored_at, wm.last_tenant_id, wm.last_statement_id)
), 0);

INSERT INTO statement_stats_hourly (hour, scope_type, scope_id, statement_count)
SELECT
  date_trunc('hour', st.stored_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC',
  s.scope_type,
  s.scope_id,
  COUNT(*)::bigint
FROM statements st
JOIN lrs l ON l.id = st.tenant_id
JOIN projects p ON p.id = l.project_id
JOIN statement_rollup_state wm ON wm.id = 1
CROSS JOIN LATERAL (
  SELECT 'global'::text AS scope_type, '*'::text AS scope_id
  UNION ALL SELECT 'organisation', p.org_id::text
  UNION ALL SELECT 'project', p.id::text
  UNION ALL SELECT 'lrs', l.id
  UNION ALL SELECT 'client', st.client_id::text WHERE st.client_id IS NOT NULL
) s
WHERE st.stored_at >= date_trunc('hour', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
  AND (st.stored_at, st.tenant_id, st.id) <= (wm.last_stored_at, wm.last_tenant_id, wm.last_statement_id)
  AND s.scope_id IS NOT NULL
  AND s.scope_id <> ''
GROUP BY 1, 2, 3
ON CONFLICT (hour, scope_type, scope_id)
DO UPDATE SET statement_count = EXCLUDED.statement_count;
