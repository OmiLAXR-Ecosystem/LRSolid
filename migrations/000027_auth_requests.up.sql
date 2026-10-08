CREATE TABLE auth_requests (
    id             text        PRIMARY KEY,
    provider_id    uuid        NOT NULL REFERENCES auth_providers(id) ON DELETE CASCADE,
    org_id         uuid        NULL REFERENCES organisations(id) ON DELETE CASCADE,
    nonce          text        NULL,
    pkce_verifier  text        NULL,
    return_to      text        NOT NULL DEFAULT '/',
    invite_token   text        NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL
);

CREATE INDEX idx_auth_requests_expires ON auth_requests(expires_at);

CREATE TABLE saml_seen_assertions (
    assertion_id text        PRIMARY KEY,
    expires_at   timestamptz NOT NULL
);

CREATE INDEX idx_saml_seen_assertions_expires ON saml_seen_assertions(expires_at);
