CREATE TABLE dashboard_configs (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     uuid        NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    user_id    uuid        REFERENCES users(id) ON DELETE CASCADE,
    scope      text        NOT NULL CHECK (scope IN ('user', 'org')),
    widgets    jsonb       NOT NULL DEFAULT '[]',
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, user_id)
);

CREATE INDEX dashboard_configs_org_idx  ON dashboard_configs (org_id);
CREATE INDEX dashboard_configs_user_idx ON dashboard_configs (org_id, user_id);
