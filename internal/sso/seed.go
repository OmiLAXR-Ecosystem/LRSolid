package sso

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/secrets"
)

// Seed creates env-configured providers for the default organisation when that
// slug is not already present, and applies LOCAL_LOGIN_ENABLED when it is set.
func Seed(ctx context.Context, db *pgxpool.Pool, store secrets.Store, cfg config.Config) error {
	if cfg.LocalLoginExplicit {
		if _, err := db.Exec(ctx, `
			UPDATE organisations SET local_login_enabled = $2, updated_at = now() WHERE slug = $1
		`, cfg.DefaultOrg, cfg.LocalLoginEnabled); err != nil {
			return err
		}
	}

	var orgID string
	err := db.QueryRow(ctx, `SELECT id::text FROM organisations WHERE slug = $1`, cfg.DefaultOrg).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if cfg.OIDCEnabled {
		if err := seedOIDC(ctx, db, store, cfg, orgID); err != nil {
			return err
		}
	}
	if cfg.SAMLEnabled {
		if err := seedSAML(ctx, db, store, cfg, orgID); err != nil {
			return err
		}
	}
	return LinkLegacyIdentities(ctx, db)
}

func providerExists(ctx context.Context, db *pgxpool.Pool, slug string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_providers WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func seedOIDC(ctx context.Context, db *pgxpool.Pool, store secrets.Store, cfg config.Config, orgID string) error {
	exists, err := providerExists(ctx, db, cfg.OIDCSlug)
	if err != nil || exists {
		return err
	}
	if cfg.OIDCIssuer == "" || cfg.OIDCClientID == "" {
		return errors.New("OIDC_ISSUER and OIDC_CLIENT_ID are required when OIDC_ENABLED=true")
	}
	scopes := strings.Fields(cfg.OIDCScopes)
	raw, err := json.Marshal(OIDCSettings{
		Issuer:               cfg.OIDCIssuer,
		ClientID:             cfg.OIDCClientID,
		Scopes:               scopes,
		RequireEmailVerified: cfg.OIDCRequireEmailVerified,
	})
	if err != nil {
		return err
	}
	secretRef, err := putSecret(ctx, store, "auth-provider/"+cfg.OIDCSlug, cfg.OIDCClientSecret)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO auth_providers (org_id, slug, type, display_name, enabled, config, secret_ref, auto_provision, default_role, link_by_email)
		VALUES ($1::uuid, $2, 'oidc', $3, true, $4::jsonb, $5, $6, $7, true)
	`, orgID, cfg.OIDCSlug, cfg.OIDCDisplayName, raw, secretRef, cfg.OIDCAutoProvision, cfg.OIDCDefaultRole)
	return err
}

func seedSAML(ctx context.Context, db *pgxpool.Pool, store secrets.Store, cfg config.Config, orgID string) error {
	exists, err := providerExists(ctx, db, cfg.SAMLSlug)
	if err != nil || exists {
		return err
	}
	if cfg.SAMLIDPMetadataURL == "" && cfg.SAMLIDPMetadataXML == "" {
		return errors.New("SAML_IDP_METADATA_URL or SAML_IDP_METADATA_XML is required when SAML_ENABLED=true")
	}
	entityID := cfg.SAMLSPEntityID
	if entityID == "" {
		entityID = SAMLMetadataURL(cfg.AppURL, cfg.SAMLSlug)
	}
	certPEM := cfg.SAMLSPCertPEM
	keyPEM := cfg.SAMLSPKeyPEM
	if certPEM == "" || keyPEM == "" {
		certPEM, keyPEM, err = GenerateSPCredentials(entityID)
		if err != nil {
			return err
		}
	}
	raw, err := json.Marshal(SAMLSettings{
		IDPMetadataURL: cfg.SAMLIDPMetadataURL,
		IDPMetadataXML: cfg.SAMLIDPMetadataXML,
		SPEntityID:     entityID,
		SPCertPEM:      certPEM,
	})
	if err != nil {
		return err
	}
	secretRef, err := putSecret(ctx, store, "auth-provider/"+cfg.SAMLSlug, keyPEM)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO auth_providers (org_id, slug, type, display_name, enabled, config, secret_ref, auto_provision, default_role, link_by_email)
		VALUES ($1::uuid, $2, 'saml', $3, true, $4::jsonb, $5, $6, $7, true)
	`, orgID, cfg.SAMLSlug, cfg.SAMLDisplayName, raw, secretRef, cfg.SAMLAutoProvision, cfg.SAMLDefaultRole)
	return err
}

func putSecret(ctx context.Context, store secrets.Store, ref, plaintext string) (*string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return nil, nil
	}
	if store == nil {
		return nil, errors.New("DATA_STORE_SECRET_KEY is required to store SSO secrets")
	}
	if err := store.Put(ctx, ref, []byte(plaintext)); err != nil {
		return nil, err
	}
	return &ref, nil
}
