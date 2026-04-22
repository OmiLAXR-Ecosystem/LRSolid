CREATE TABLE plugins (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       uuid        NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    slug         text        NOT NULL,
    name         text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    s3_key       text        NOT NULL,
    source       text        NOT NULL,
    enabled      bool        NOT NULL DEFAULT true,
    installed_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    installed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, slug)
);

CREATE INDEX plugins_org_idx ON plugins (org_id, enabled);
