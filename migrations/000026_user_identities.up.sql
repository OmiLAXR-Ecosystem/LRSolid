CREATE TABLE user_identities (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id   uuid        NOT NULL REFERENCES auth_providers(id) ON DELETE CASCADE,
    subject       text        NOT NULL,
    email         text        NULL,
    attributes    jsonb       NOT NULL DEFAULT '{}',
    last_login_at timestamptz NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_id, subject)
);

CREATE INDEX idx_user_identities_user ON user_identities(user_id);

-- Legacy users.auth_provider stored a provider slug. Copy rows once a matching
-- provider exists; at migration time the table is empty, and startup seeding
-- repeats this insert.
INSERT INTO user_identities (user_id, provider_id, subject, email)
SELECT u.id, p.id, u.auth_subject, u.email
FROM users u
JOIN auth_providers p ON p.slug = u.auth_provider
WHERE u.auth_subject IS NOT NULL
ON CONFLICT (provider_id, subject) DO NOTHING;
