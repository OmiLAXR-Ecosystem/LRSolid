CREATE TABLE auth_providers (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          uuid        NULL REFERENCES organisations(id) ON DELETE CASCADE,
    slug            text        NOT NULL,
    type            text        NOT NULL CHECK (type IN ('oidc', 'saml')),
    display_name    text        NOT NULL,
    enabled         boolean     NOT NULL DEFAULT true,
    config          jsonb       NOT NULL DEFAULT '{}',
    secret_ref      text        NULL,
    auto_provision  boolean     NOT NULL DEFAULT false,
    default_role    text        NOT NULL DEFAULT 'viewer' CHECK (default_role IN ('owner', 'admin', 'member', 'viewer')),
    link_by_email   boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug)
);

CREATE INDEX idx_auth_providers_org ON auth_providers(org_id);
