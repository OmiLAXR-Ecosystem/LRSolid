ALTER TABLE org_export_settings
  DROP CONSTRAINT IF EXISTS chk_org_export_settings_default_target_type;

ALTER TABLE org_export_settings
  ADD CONSTRAINT chk_org_export_settings_default_target_type
  CHECK (default_target_type IN ('local','rustfs','s3','gdrive','onedrive','icloud'));
