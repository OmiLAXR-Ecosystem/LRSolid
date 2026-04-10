CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS data_stores (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name          text        NOT NULL,
  database_type text        NOT NULL,
  host          text        NOT NULL,
  port          integer     NULL,
  database_name text        NULL,
  description   text        NULL,
  is_default    boolean     NOT NULL DEFAULT false,
  is_active     boolean     NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_data_stores_database_type CHECK (database_type IN ('postgres', 'mongodb'))
);

CREATE INDEX IF NOT EXISTS idx_data_stores_default ON data_stores(is_default);
CREATE INDEX IF NOT EXISTS idx_data_stores_type ON data_stores(database_type);

INSERT INTO data_stores (name, database_type, host, port, database_name, description, is_default)
SELECT
  'Primary Postgres (Control Plane Default)',
  'postgres',
  COALESCE(inet_server_addr()::text, '127.0.0.1'),
  inet_server_port(),
  current_database(),
  'Default seeded data store pointing to the current control-plane Postgres database.',
  true
WHERE NOT EXISTS (
  SELECT 1 FROM data_stores WHERE is_default = true
);

ALTER TABLE lrs
  ADD COLUMN IF NOT EXISTS data_store_id uuid NULL REFERENCES data_stores(id) ON DELETE RESTRICT;

WITH default_store AS (
  SELECT id FROM data_stores WHERE is_default = true ORDER BY created_at ASC LIMIT 1
)
UPDATE lrs
SET data_store_id = default_store.id
FROM default_store
WHERE lrs.data_store_id IS NULL;

ALTER TABLE lrs
  ALTER COLUMN data_store_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_lrs_data_store ON lrs(data_store_id);
