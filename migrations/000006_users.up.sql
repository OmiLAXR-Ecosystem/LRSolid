CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
                                     id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL UNIQUE,
    display_name  text        NULL,

    -- for local auth (optional). If you use external auth, keep it NULL.
    password_hash text        NULL,

    -- for SSO / external identity (optional)
    auth_provider text        NULL,
    auth_subject  text        NULL,

    created_at    timestamptz NOT NULL DEFAULT now(),
    disabled_at   timestamptz NULL
    );

CREATE INDEX IF NOT EXISTS idx_users_auth ON users(auth_provider, auth_subject);