package sso

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionUser is the account a completed SSO login should sign in as.
type SessionUser struct {
	ID    string
	Email string
	Role  string
}

// Resolve finds or creates the LRSolid user for an external identity.
func Resolve(ctx context.Context, db *pgxpool.Pool, provider Provider, identity Identity, targetOrgID, inviteToken string, roleMapping map[string]string, requireVerified bool) (SessionUser, error) {
	identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
	identity.Subject = strings.TrimSpace(identity.Subject)
	if identity.Subject == "" {
		return SessionUser{}, ErrTransientSubject
	}
	if requireVerified && identity.Email != "" && !identity.EmailVerified {
		return SessionUser{}, ErrEmailUnverified
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return SessionUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	userID, err := findIdentityUser(ctx, tx, provider.ID, identity.Subject)
	if err != nil {
		return SessionUser{}, err
	}
	if userID == "" && provider.LinkByEmail && identity.Email != "" && identity.EmailVerified {
		userID, err = findUserByEmail(ctx, tx, identity.Email)
		if err != nil {
			return SessionUser{}, err
		}
	}

	inviteRole := ""
	inviteID := ""
	if userID == "" {
		inviteID, inviteRole, err = findInvite(ctx, tx, targetOrgID, identity.Email, inviteToken)
		if err != nil {
			return SessionUser{}, err
		}
	}

	created := false
	if userID == "" {
		if inviteID == "" && !provider.AutoProvision {
			return SessionUser{}, ErrNoAccount
		}
		if identity.Email == "" {
			return SessionUser{}, ErrEmailRequired
		}
		userID, err = insertUser(ctx, tx, identity, provider.Slug)
		if err != nil {
			return SessionUser{}, err
		}
		created = true
	}

	if err := assertEnabled(ctx, tx, userID); err != nil {
		return SessionUser{}, err
	}
	if err := upsertIdentity(ctx, tx, userID, provider, identity); err != nil {
		return SessionUser{}, err
	}

	role, sync := MapRole(identity.Groups, roleMapping, provider.DefaultRole)
	if inviteRole != "" && !sync {
		role = inviteRole
	}
	if roleRank(role) == 0 {
		role = "viewer"
	}
	if err := ensureMembership(ctx, tx, targetOrgID, userID, role, sync || created); err != nil {
		return SessionUser{}, err
	}
	if inviteID != "" {
		if _, err := tx.Exec(ctx, `UPDATE user_invites SET accepted_at = now() WHERE id = $1::uuid AND accepted_at IS NULL`, inviteID); err != nil {
			return SessionUser{}, err
		}
	}

	var email, membershipRole string
	err = tx.QueryRow(ctx, `
		SELECT u.email, om.role
		FROM users u
		JOIN org_memberships om ON om.user_id = u.id AND om.org_id = $2::uuid
		WHERE u.id = $1::uuid
	`, userID, targetOrgID).Scan(&email, &membershipRole)
	if err != nil {
		return SessionUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionUser{}, err
	}
	return SessionUser{ID: userID, Email: email, Role: membershipRole}, nil
}

func findIdentityUser(ctx context.Context, tx pgx.Tx, providerID, subject string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT user_id::text FROM user_identities WHERE provider_id = $1::uuid AND subject = $2
	`, providerID, subject).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func findUserByEmail(ctx context.Context, tx pgx.Tx, email string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE lower(email) = $1`, email).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func findInvite(ctx context.Context, tx pgx.Tx, orgID, email, token string) (id, role string, err error) {
	if token != "" {
		err = tx.QueryRow(ctx, `
			SELECT id::text, role
			FROM user_invites
			WHERE token = $1 AND org_id = $2::uuid AND accepted_at IS NULL AND expires_at > now()
			  AND ($3 = '' OR lower(email) = $3)
		`, token, orgID, email).Scan(&id, &role)
		if err == nil {
			return id, role, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", "", err
		}
	}
	if email == "" {
		return "", "", nil
	}
	err = tx.QueryRow(ctx, `
		SELECT id::text, role
		FROM user_invites
		WHERE org_id = $1::uuid AND lower(email) = $2 AND accepted_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID, email).Scan(&id, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	return id, role, err
}

func insertUser(ctx context.Context, tx pgx.Tx, identity Identity, providerSlug string) (string, error) {
	var display *string
	if identity.DisplayName != "" {
		display = &identity.DisplayName
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, auth_provider, auth_subject)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, identity.Email, display, providerSlug, identity.Subject).Scan(&id)
	return id, err
}

func assertEnabled(ctx context.Context, tx pgx.Tx, userID string) error {
	var disabled *time.Time
	if err := tx.QueryRow(ctx, `SELECT disabled_at FROM users WHERE id = $1::uuid`, userID).Scan(&disabled); err != nil {
		return err
	}
	if disabled != nil {
		return ErrDisabled
	}
	return nil
}

func upsertIdentity(ctx context.Context, tx pgx.Tx, userID string, provider Provider, identity Identity) error {
	raw, err := json.Marshal(identity.Raw)
	if err != nil || string(raw) == "null" {
		raw = []byte(`{}`)
	}
	var email *string
	if identity.Email != "" {
		email = &identity.Email
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_identities (user_id, provider_id, subject, email, attributes, last_login_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::jsonb, now())
		ON CONFLICT (provider_id, subject) DO UPDATE SET
		    user_id = EXCLUDED.user_id,
		    email = EXCLUDED.email,
		    attributes = EXCLUDED.attributes,
		    last_login_at = now()
	`, userID, provider.ID, identity.Subject, email, raw)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET auth_provider = $2,
		    auth_subject = $3,
		    display_name = COALESCE(display_name, NULLIF($4, ''))
		WHERE id = $1::uuid
	`, userID, provider.Slug, identity.Subject, identity.DisplayName)
	return err
}

func ensureMembership(ctx context.Context, tx pgx.Tx, orgID, userID, role string, overwrite bool) error {
	if overwrite {
		_, err := tx.Exec(ctx, `
			INSERT INTO org_memberships (org_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3)
			ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, orgID, userID, role)
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO org_memberships (org_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (org_id, user_id) DO NOTHING
	`, orgID, userID, role)
	return err
}
