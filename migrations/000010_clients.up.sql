CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Clients are auth entities (key/secret) typically used for xAPI Basic Auth.
-- secret_hash stores a hash (never store the raw secret).
-- key_id can be used as username.

CREATE TABLE IF NOT EXISTS clients (
                                       id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    lrs_id       text        NULL REFERENCES lrs(id) ON DELETE SET NULL, -- optional: restrict client to one LRS

    name         text        NOT NULL,
    key_id       text        NOT NULL UNIQUE,
    secret_hash  text        NOT NULL,

    -- e.g. {"statements:write":true,"statements:read":true,"state:write":true}
    scopes       jsonb       NOT NULL DEFAULT '{}'::jsonb,

    created_at   timestamptz NOT NULL DEFAULT now(),
    revoked_at   timestamptz NULL
    );

CREATE INDEX IF NOT EXISTS idx_clients_project ON clients(project_id);
CREATE INDEX IF NOT EXISTS idx_clients_lrs ON clients(lrs_id);

-- Optional: query scopes via GIN if needed
-- CREATE INDEX IF NOT EXISTS idx_clients_scopes_gin ON clients USING gin (scopes jsonb_path_ops);