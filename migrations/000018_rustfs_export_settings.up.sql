ALTER TABLE org_export_settings
  ADD COLUMN IF NOT EXISTS default_target_type text NOT NULL DEFAULT 's3',
  ADD COLUMN IF NOT EXISTS s3_upload_base_url text NULL,
  ADD COLUMN IF NOT EXISTS s3_public_base_url text NULL,
  ADD COLUMN IF NOT EXISTS s3_auth_token text NULL,
  ADD COLUMN IF NOT EXISTS s3_path_prefix text NOT NULL DEFAULT 'exports';

ALTER TABLE org_export_settings
  DROP CONSTRAINT IF EXISTS chk_org_export_settings_default_target_type;

ALTER TABLE org_export_settings
  ADD CONSTRAINT chk_org_export_settings_default_target_type
  CHECK (default_target_type IN ('local','s3','gdrive','onedrive','icloud'));

UPDATE org_export_settings
SET default_target_type = 's3'
WHERE default_target_type IS NULL OR default_target_type = '' OR default_target_type = 'rustfs';

ALTER TABLE export_jobs
  DROP CONSTRAINT IF EXISTS chk_export_jobs_target_type;

ALTER TABLE export_jobs
  ADD CONSTRAINT chk_export_jobs_target_type
  CHECK (target_type IS NULL OR target_type IN ('s3','gdrive','onedrive','icloud'));
