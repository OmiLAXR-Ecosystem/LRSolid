-- Add connection_string for external data stores (MongoDB, external Postgres, etc.)
-- org_id is optional: NULL = platform-managed store, set = org-owned (BYOD) store.
ALTER TABLE data_stores
  ADD COLUMN IF NOT EXISTS connection_string text NULL,
  ADD COLUMN IF NOT EXISTS org_id uuid NULL REFERENCES organisations(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_data_stores_org ON data_stores(org_id);
