-- LRS are the "tenants" referenced by tenant_id in statements/state_docs/profiles/attachments.
-- Keep id as TEXT to match existing tables.

CREATE TABLE IF NOT EXISTS lrs (
                                   id          text        PRIMARY KEY, -- equals tenant_id
                                   project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
    );

CREATE INDEX IF NOT EXISTS idx_lrs_project ON lrs(project_id);

-- Optional fine-grained access override per LRS
CREATE TABLE IF NOT EXISTS lrs_memberships (
                                               lrs_id     text        NOT NULL REFERENCES lrs(id) ON DELETE CASCADE,
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (lrs_id, user_id),
    CONSTRAINT chk_lrs_role CHECK (role IN ('admin','writer','reader'))
    );

CREATE INDEX IF NOT EXISTS idx_lrs_memberships_user ON lrs_memberships(user_id);