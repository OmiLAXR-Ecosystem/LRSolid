DROP INDEX IF EXISTS idx_client_allowed_users_user;
DROP TABLE IF EXISTS client_allowed_users;

ALTER TABLE clients
  DROP COLUMN IF EXISTS expires_at,
  DROP COLUMN IF EXISTS note;
