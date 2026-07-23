ALTER TABLE org_export_settings
  ADD COLUMN IF NOT EXISTS s3_access_key text NULL,
  ADD COLUMN IF NOT EXISTS s3_secret_key text NULL,
  ADD COLUMN IF NOT EXISTS s3_region text NOT NULL DEFAULT 'us-east-1';

-- Exports are no longer exposed via a public URL; downloads are proxied
-- through the API using authenticated S3 requests.
ALTER TABLE org_export_settings
  DROP COLUMN IF EXISTS s3_public_base_url;
