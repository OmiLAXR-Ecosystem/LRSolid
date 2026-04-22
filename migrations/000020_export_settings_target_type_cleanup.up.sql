UPDATE org_export_settings
SET default_target_type = 's3'
WHERE COALESCE(TRIM(default_target_type), '') = '' OR default_target_type = 'rustfs';

ALTER TABLE org_export_settings
  DROP CONSTRAINT IF EXISTS chk_org_export_settings_default_target_type;

ALTER TABLE org_export_settings
  ADD CONSTRAINT chk_org_export_settings_default_target_type
  CHECK (default_target_type IN ('local','s3','gdrive','onedrive','icloud'));
