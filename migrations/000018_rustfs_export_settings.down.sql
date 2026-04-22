ALTER TABLE export_jobs
  DROP CONSTRAINT IF EXISTS chk_export_jobs_target_type;

ALTER TABLE export_jobs
  ADD CONSTRAINT chk_export_jobs_target_type
  CHECK (target_type IS NULL OR target_type IN ('s3','gdrive','onedrive','icloud'));

ALTER TABLE org_export_settings
  DROP CONSTRAINT IF EXISTS chk_org_export_settings_default_target_type;

ALTER TABLE org_export_settings
  DROP COLUMN IF EXISTS default_target_type,
  DROP COLUMN IF EXISTS s3_upload_base_url,
  DROP COLUMN IF EXISTS s3_public_base_url,
  DROP COLUMN IF EXISTS s3_auth_token,
  DROP COLUMN IF EXISTS s3_path_prefix;
