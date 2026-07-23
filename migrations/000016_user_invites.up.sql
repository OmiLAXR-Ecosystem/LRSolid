CREATE TABLE user_invites (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token       TEXT        NOT NULL UNIQUE,
    org_id      UUID        NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    email       TEXT        NOT NULL,
    role        TEXT        NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    invited_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ
);
