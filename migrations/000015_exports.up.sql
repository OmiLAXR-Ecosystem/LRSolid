CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS org_export_settings (
  org_id                             uuid PRIMARY KEY REFERENCES organisations(id) ON DELETE CASCADE,
  require_remote_target_above_bytes  bigint NOT NULL DEFAULT 2147483648,
  local_max_bytes                    bigint NOT NULL DEFAULT 2147483648,
  updated_at                         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_org_export_settings_threshold_non_negative
    CHECK (require_remote_target_above_bytes >= 0 AND local_max_bytes >= 0)
);

CREATE TABLE IF NOT EXISTS export_jobs (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id           uuid NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
  created_by_user  uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status           text NOT NULL DEFAULT 'queued',
  filters          jsonb NOT NULL DEFAULT '{}'::jsonb,
  format           text NOT NULL DEFAULT 'ndjson',
  compression      text NOT NULL DEFAULT 'gzip',
  estimated_rows   bigint NOT NULL DEFAULT 0,
  estimated_bytes  bigint NOT NULL DEFAULT 0,
  target_type      text NULL,
  target_config    jsonb NULL,
  artifact_ref     text NULL,
  error_message    text NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  started_at       timestamptz NULL,
  finished_at      timestamptz NULL,
  CONSTRAINT chk_export_jobs_status CHECK (status IN ('queued','running','completed','failed','canceled')),
  CONSTRAINT chk_export_jobs_format CHECK (format IN ('ndjson')),
  CONSTRAINT chk_export_jobs_compression CHECK (compression IN ('none','gzip')),
  CONSTRAINT chk_export_jobs_target_type CHECK (target_type IS NULL OR target_type IN ('s3','gdrive','onedrive','icloud'))
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_org_created ON export_jobs(org_id, created_at DESC);
