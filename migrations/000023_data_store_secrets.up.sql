CREATE TABLE IF NOT EXISTS secrets (
  ref         text        PRIMARY KEY,
  ciphertext  bytea       NOT NULL,
  nonce       bytea       NOT NULL,
  key_version integer     NOT NULL DEFAULT 1,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE data_stores
  ADD COLUMN IF NOT EXISTS secret_ref text NULL REFERENCES secrets(ref) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_data_stores_secret_ref ON data_stores(secret_ref);
