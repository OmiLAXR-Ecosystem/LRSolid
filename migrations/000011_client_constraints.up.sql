ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS note text NULL,
  ADD COLUMN IF NOT EXISTS expires_at timestamptz NULL;

CREATE TABLE IF NOT EXISTS client_allowed_users (
  client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (client_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_client_allowed_users_user ON client_allowed_users(user_id);
