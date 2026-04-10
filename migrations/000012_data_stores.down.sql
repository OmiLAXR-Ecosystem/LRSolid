DROP INDEX IF EXISTS idx_lrs_data_store;

ALTER TABLE lrs
  DROP COLUMN IF EXISTS data_store_id;

DROP INDEX IF EXISTS idx_data_stores_type;
DROP INDEX IF EXISTS idx_data_stores_default;
DROP TABLE IF EXISTS data_stores;
