DROP INDEX IF EXISTS idx_data_stores_secret_ref;

ALTER TABLE data_stores
  DROP COLUMN IF EXISTS secret_ref;

DROP TABLE IF EXISTS secrets;
