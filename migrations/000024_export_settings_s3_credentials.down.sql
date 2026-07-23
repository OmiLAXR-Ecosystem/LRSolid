ALTER TABLE org_export_settings
  ADD COLUMN IF NOT EXISTS s3_public_base_url text NULL;

ALTER TABLE org_export_settings
  DROP COLUMN IF EXISTS s3_access_key,
  DROP COLUMN IF EXISTS s3_secret_key,
  DROP COLUMN IF EXISTS s3_region;
