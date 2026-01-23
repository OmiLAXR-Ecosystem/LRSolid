-- Roles: keep as TEXT + CHECK constraints (easy to extend later)

CREATE TABLE IF NOT EXISTS org_memberships (
                                               org_id     uuid        NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id),
    CONSTRAINT chk_org_role CHECK (role IN ('owner','admin','member','viewer'))
    );

CREATE INDEX IF NOT EXISTS idx_org_memberships_user ON org_memberships(user_id);

CREATE TABLE IF NOT EXISTS project_memberships (
                                                   project_id uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id),
    CONSTRAINT chk_project_role CHECK (role IN ('owner','admin','member','viewer'))
    );

CREATE INDEX IF NOT EXISTS idx_project_memberships_user ON project_memberships(user_id);