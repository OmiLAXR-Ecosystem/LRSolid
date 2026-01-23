CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS projects (
                                        id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      uuid        NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,

    slug        text        NOT NULL,
    name        text        NOT NULL,
    description text        NULL,

    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    UNIQUE (org_id, slug)
    );

CREATE INDEX IF NOT EXISTS idx_projects_org ON projects(org_id);