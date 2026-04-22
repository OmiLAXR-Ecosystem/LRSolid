ALTER TABLE org_export_settings
  ADD COLUMN IF NOT EXISTS s3_upload_base_url text NULL,
  ADD COLUMN IF NOT EXISTS s3_public_base_url text NULL,
  ADD COLUMN IF NOT EXISTS s3_auth_token text NULL,
  ADD COLUMN IF NOT EXISTS s3_path_prefix text NOT NULL DEFAULT 'exports';

UPDATE org_export_settings
SET s3_path_prefix = 'exports'
WHERE COALESCE(TRIM(s3_path_prefix), '') = '';
