package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const providerSelect = `
	SELECT p.id::text, p.org_id::text, COALESCE(o.slug, ''), p.slug, p.type, p.display_name,
	       p.enabled, p.config, p.secret_ref, p.auto_provision, p.default_role, p.link_by_email
	FROM auth_providers p
	LEFT JOIN organisations o ON o.id = p.org_id
`

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func scanProvider(row pgx.Row) (Provider, error) {
	var p Provider
	var orgID *string
	var secret *string
	var config []byte
	err := row.Scan(
		&p.ID, &orgID, &p.OrgSlug, &p.Slug, &p.Type, &p.DisplayName,
		&p.Enabled, &config, &secret, &p.AutoProvision, &p.DefaultRole, &p.LinkByEmail,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrNotFound
	}
	if err != nil {
		return Provider{}, err
	}
	p.OrgID = orgID
	p.SecretRef = secret
	if len(config) == 0 {
		config = []byte(`{}`)
	}
	p.Config = config
	return p, nil
}

// LoadBySlug returns an enabled provider. orgSlug prefers that organisation's
// provider, then a provider with no organisation.
func LoadBySlug(ctx context.Context, db *pgxpool.Pool, slug, orgSlug string) (Provider, error) {
	row := db.QueryRow(ctx, providerSelect+`
		WHERE p.slug = $1
		  AND p.enabled
		  AND ($2 = '' OR o.slug = $2 OR p.org_id IS NULL)
		ORDER BY CASE WHEN o.slug = $2 THEN 0 WHEN p.org_id IS NULL THEN 1 ELSE 2 END
		LIMIT 1
	`, slug, orgSlug)
	return scanProvider(row)
}

// LoadAnyBySlug returns a provider even when it is disabled.
func LoadAnyBySlug(ctx context.Context, db *pgxpool.Pool, slug string) (Provider, error) {
	return scanProvider(db.QueryRow(ctx, providerSelect+` WHERE p.slug = $1`, slug))
}

// LoadByID returns a provider regardless of enabled state.
func LoadByID(ctx context.Context, db *pgxpool.Pool, id string) (Provider, error) {
	return scanProvider(db.QueryRow(ctx, providerSelect+` WHERE p.id = $1::uuid`, id))
}

// ListLoginProviders returns enabled providers for the sign-in page.
func ListLoginProviders(ctx context.Context, db *pgxpool.Pool, orgSlug string) ([]LoginProvider, error) {
	rows, err := db.Query(ctx, `
		SELECT p.slug, p.type, p.display_name
		FROM auth_providers p
		LEFT JOIN organisations o ON o.id = p.org_id
		WHERE p.enabled
		  AND ($1 = '' OR o.slug = $1 OR p.org_id IS NULL)
		ORDER BY p.display_name
	`, orgSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LoginProvider{}
	for rows.Next() {
		var p LoginProvider
		if err := rows.Scan(&p.Slug, &p.Type, &p.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LocalLoginEnabled reports whether the organisation still accepts passwords.
// Unknown organisations stay enabled so a missing org cannot lock the form.
func LocalLoginEnabled(ctx context.Context, db *pgxpool.Pool, orgSlug string) (bool, error) {
	var enabled bool
	err := db.QueryRow(ctx, `
		SELECT local_login_enabled FROM organisations WHERE slug = $1
	`, orgSlug).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	return enabled, nil
}

func SaveRequest(ctx context.Context, db *pgxpool.Pool, req AuthRequest) error {
	_, err := db.Exec(ctx, `
		INSERT INTO auth_requests (id, provider_id, org_id, nonce, pkce_verifier, return_to, invite_token, expires_at)
		VALUES ($1, $2::uuid, $3::uuid, NULLIF($4, ''), NULLIF($5, ''), $6, NULLIF($7, ''), $8)
	`, req.ID, req.ProviderID, req.OrgID, req.Nonce, req.PKCEVerifier, req.ReturnTo, req.InviteToken, req.ExpiresAt)
	return err
}

// TakeRequest deletes and returns a pending login request.
func TakeRequest(ctx context.Context, db *pgxpool.Pool, id string) (AuthRequest, error) {
	var req AuthRequest
	var orgID *string
	var nonce, verifier, invite *string
	err := db.QueryRow(ctx, `
		DELETE FROM auth_requests
		WHERE id = $1
		RETURNING id, provider_id::text, org_id::text, nonce, pkce_verifier, return_to, invite_token, expires_at
	`, id).Scan(&req.ID, &req.ProviderID, &orgID, &nonce, &verifier, &req.ReturnTo, &invite, &req.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthRequest{}, ErrNotFound
	}
	if err != nil {
		return AuthRequest{}, err
	}
	req.OrgID = orgID
	if nonce != nil {
		req.Nonce = *nonce
	}
	if verifier != nil {
		req.PKCEVerifier = *verifier
	}
	if invite != nil {
		req.InviteToken = *invite
	}
	if time.Now().After(req.ExpiresAt) {
		return AuthRequest{}, ErrExpired
	}
	req.ReturnTo = SafeReturnTo(req.ReturnTo)
	return req, nil
}

// SeeAssertion records a SAML assertion ID. A duplicate is a replay.
func SeeAssertion(ctx context.Context, db *pgxpool.Pool, assertionID string, expires time.Time) error {
	if assertionID == "" {
		return nil
	}
	if expires.IsZero() || expires.Before(time.Now()) {
		expires = time.Now().Add(24 * time.Hour)
	}
	tag, err := db.Exec(ctx, `
		INSERT INTO saml_seen_assertions (assertion_id, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (assertion_id) DO NOTHING
	`, assertionID, expires)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReplay
	}
	return nil
}

// Cleanup drops expired login requests and seen assertions.
func Cleanup(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `DELETE FROM auth_requests WHERE expires_at < now()`); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `DELETE FROM saml_seen_assertions WHERE expires_at < now()`)
	return err
}

// LinkLegacyIdentities copies users.auth_provider/auth_subject once a provider
// with that slug exists.
func LinkLegacyIdentities(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		INSERT INTO user_identities (user_id, provider_id, subject, email)
		SELECT u.id, p.id, u.auth_subject, u.email
		FROM users u
		JOIN auth_providers p ON p.slug = u.auth_provider
		WHERE u.auth_subject IS NOT NULL
		ON CONFLICT (provider_id, subject) DO NOTHING
	`)
	return err
}
