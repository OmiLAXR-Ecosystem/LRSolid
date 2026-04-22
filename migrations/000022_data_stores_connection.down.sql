DROP INDEX IF EXISTS idx_data_stores_org;

ALTER TABLE data_stores
  DROP COLUMN IF EXISTS org_id,
  DROP COLUMN IF EXISTS connection_string;
