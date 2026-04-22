ALTER TABLE org_export_settings
  DROP COLUMN IF EXISTS s3_upload_base_url,
  DROP COLUMN IF EXISTS s3_public_base_url,
  DROP COLUMN IF EXISTS s3_auth_token,
  DROP COLUMN IF EXISTS s3_path_prefix;
