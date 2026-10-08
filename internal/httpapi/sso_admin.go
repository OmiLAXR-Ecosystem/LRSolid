package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/sso"
)

var providerSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type authProviderResponse struct {
	ID            string          `json:"id"`
	Slug          string          `json:"slug"`
	Type          string          `json:"type"`
	DisplayName   string          `json:"displayName"`
	Enabled       bool            `json:"enabled"`
	AutoProvision bool            `json:"autoProvision"`
	DefaultRole   string          `json:"defaultRole"`
	LinkByEmail   bool            `json:"linkByEmail"`
	HasSecret     bool            `json:"hasSecret"`
	Config        json.RawMessage `json:"config"`
}

type authProviderWrite struct {
	Slug            string          `json:"slug"`
	Type            string          `json:"type"`
	DisplayName     string          `json:"displayName"`
	Enabled         *bool           `json:"enabled"`
	AutoProvision   *bool           `json:"autoProvision"`
	DefaultRole     string          `json:"defaultRole"`
	LinkByEmail     *bool           `json:"linkByEmail"`
	Config          json.RawMessage `json:"config"`
	ClientSecret    string          `json:"clientSecret"`
	SPPrivateKeyPEM string          `json:"spPrivateKeyPem"`
	GenerateSPKey   bool            `json:"generateSpKey"`
}

type authSettingsResponse struct {
	LocalLoginEnabled bool `json:"localLoginEnabled"`
}

func ListAuthProviders(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		rows, err := db.Query(r.Context(), `
			SELECT id::text, slug, type, display_name, enabled, auto_provision, default_role, link_by_email, secret_ref IS NOT NULL, config
			FROM auth_providers
			WHERE org_id = $1::uuid
			ORDER BY display_name
		`, orgID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list providers failed"})
			return
		}
		defer rows.Close()
		items := []authProviderResponse{}
		for rows.Next() {
			item, err := scanAuthProvider(rows)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list providers failed"})
				return
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func CreateAuthProvider(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		_, req, ok := readProviderWrite(w, r)
		if !ok {
			return
		}
		if err := validateProviderWrite(req, true); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		configJSON, secret, err := normalizeProviderSecrets(req, nil, cfg)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		var secretRef *string
		if secret != "" {
			ref := "auth-provider/" + req.Slug
			if err := putProviderSecret(r.Context(), store, ref, secret); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
				return
			}
			secretRef = &ref
		}
		var id string
		err = db.QueryRow(r.Context(), `
			INSERT INTO auth_providers (org_id, slug, type, display_name, enabled, config, secret_ref, auto_provision, default_role, link_by_email)
			VALUES ($1::uuid, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10)
			RETURNING id::text
		`, orgID, req.Slug, req.Type, req.DisplayName, boolOr(req.Enabled, true), configJSON, secretRef, boolOr(req.AutoProvision, false), roleOr(req.DefaultRole), boolOr(req.LinkByEmail, false)).Scan(&id)
		if err != nil {
			writeProviderStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

func UpdateAuthProvider(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		current, ok := providerInOrg(w, r, db, orgID)
		if !ok {
			return
		}
		_, req, ok := readProviderWrite(w, r)
		if !ok {
			return
		}
		if req.Type == "" {
			req.Type = current.Type
		}
		if req.Slug == "" {
			req.Slug = current.Slug
		}
		if err := validateProviderWrite(req, false); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		configJSON, secret, err := normalizeProviderSecrets(req, &current, cfg)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		secretRef := current.SecretRef
		if secret != "" {
			ref := "auth-provider/" + req.Slug
			if current.SecretRef != nil && *current.SecretRef != "" {
				ref = *current.SecretRef
			}
			if err := putProviderSecret(r.Context(), store, ref, secret); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
				return
			}
			secretRef = &ref
		}
		_, err = db.Exec(r.Context(), `
			UPDATE auth_providers
			SET slug = $3, type = $4, display_name = $5, enabled = $6, config = $7::jsonb,
			    secret_ref = $8, auto_provision = $9, default_role = $10, link_by_email = $11, updated_at = now()
			WHERE id = $1::uuid AND org_id = $2::uuid
		`, current.ID, orgID, req.Slug, req.Type, req.DisplayName, boolOr(req.Enabled, current.Enabled), configJSON, secretRef, boolOr(req.AutoProvision, current.AutoProvision), roleOr(req.DefaultRole), boolOr(req.LinkByEmail, current.LinkByEmail))
		if err != nil {
			writeProviderStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": current.ID})
	}
}

func DeleteAuthProvider(db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		current, ok := providerInOrg(w, r, db, orgID)
		if !ok {
			return
		}
		if _, err := db.Exec(r.Context(), `DELETE FROM auth_providers WHERE id = $1::uuid AND org_id = $2::uuid`, current.ID, orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete provider failed"})
			return
		}
		if store != nil && current.SecretRef != nil && *current.SecretRef != "" {
			_ = store.Delete(r.Context(), *current.SecretRef)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func TestAuthProvider(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		current, ok := providerInOrg(w, r, db, orgID)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		var detail string
		var err error
		switch current.Type {
		case sso.TypeOIDC:
			detail, err = sso.TestOIDC(ctx, current.Config)
		case sso.TypeSAML:
			detail, err = sso.TestSAML(ctx, current.Config)
		default:
			err = sso.ErrInvalidProvider
		}
		if err != nil {
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "detail": detail})
	}
}

func GetAuthSettings(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		var enabled bool
		if err := db.QueryRow(r.Context(), `SELECT local_login_enabled FROM organisations WHERE id = $1::uuid`, orgID).Scan(&enabled); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load auth settings failed"})
			return
		}
		writeJSON(w, http.StatusOK, authSettingsResponse{LocalLoginEnabled: enabled})
	}
}

func PutAuthSettings(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		orgID, ok := orgIDFromRequest(w, r, db)
		if !ok {
			return
		}
		var req authSettingsResponse
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		if _, err := db.Exec(r.Context(), `
			UPDATE organisations SET local_login_enabled = $2, updated_at = now() WHERE id = $1::uuid
		`, orgID, req.LocalLoginEnabled); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "save auth settings failed"})
			return
		}
		writeJSON(w, http.StatusOK, req)
	}
}

func orgIDFromRequest(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) (string, bool) {
	orgID, err := getOrgID(r.Context(), db, chi.URLParam(r, "org"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
		return "", false
	}
	if orgID == "" {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "org not found"})
		return "", false
	}
	return orgID, true
}

func providerInOrg(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, orgID string) (sso.Provider, bool) {
	p, err := sso.LoadByID(r.Context(), db, chi.URLParam(r, "id"))
	if errors.Is(err, sso.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "provider not found"})
		return sso.Provider{}, false
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load provider failed"})
		return sso.Provider{}, false
	}
	if p.OrgID == nil || *p.OrgID != orgID {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "provider not found"})
		return sso.Provider{}, false
	}
	return p, true
}

func readProviderWrite(w http.ResponseWriter, r *http.Request) ([]byte, authProviderWrite, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid body"})
		return nil, authProviderWrite{}, false
	}
	var req authProviderWrite
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return nil, authProviderWrite{}, false
	}
	req.Slug = strings.TrimSpace(req.Slug)
	req.Type = strings.TrimSpace(req.Type)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.DefaultRole = strings.TrimSpace(req.DefaultRole)
	return body, req, true
}

func validateProviderWrite(req authProviderWrite, creating bool) error {
	if creating && (req.Slug == "" || req.DisplayName == "" || req.Type == "") {
		return errors.New("slug, type and displayName are required")
	}
	if req.Slug != "" && !providerSlugPattern.MatchString(req.Slug) {
		return errors.New("slug must be lowercase letters, numbers and dashes")
	}
	if req.Type != "" && req.Type != sso.TypeOIDC && req.Type != sso.TypeSAML {
		return errors.New("type must be oidc or saml")
	}
	if req.DefaultRole != "" && !isValidOrgRole(req.DefaultRole) {
		return errors.New("defaultRole must be owner, admin, member, or viewer")
	}
	return nil
}

func normalizeProviderSecrets(req authProviderWrite, current *sso.Provider, cfg config.Config) (json.RawMessage, string, error) {
	raw := req.Config
	if len(raw) == 0 && current != nil {
		raw = current.Config
	}
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	secret := strings.TrimSpace(req.ClientSecret)
	switch req.Type {
	case sso.TypeOIDC:
		settings, err := sso.ParseOIDC(raw)
		if err != nil {
			return nil, "", errors.New("invalid oidc config")
		}
		if settings.Issuer == "" || settings.ClientID == "" {
			return nil, "", errors.New("issuer and clientId are required")
		}
		encoded, err := json.Marshal(settings)
		return encoded, secret, err
	case sso.TypeSAML:
		settings, err := sso.ParseSAML(raw)
		if err != nil {
			return nil, "", errors.New("invalid saml config")
		}
		if strings.TrimSpace(settings.IDPMetadataURL) == "" && strings.TrimSpace(settings.IDPMetadataXML) == "" {
			return nil, "", errors.New("idpMetadataUrl or idpMetadataXml is required")
		}
		if settings.SPEntityID == "" {
			settings.SPEntityID = sso.SAMLMetadataURL(cfg.AppURL, req.Slug)
		}
		if settings.SPCertPEM == "" && current != nil {
			if prev, err := sso.ParseSAML(current.Config); err == nil {
				settings.SPCertPEM = prev.SPCertPEM
			}
		}
		key := strings.TrimSpace(req.SPPrivateKeyPEM)
		if key != "" {
			secret = key
		}
		if req.GenerateSPKey || settings.SPCertPEM == "" || (secret == "" && (current == nil || current.SecretRef == nil)) {
			cert, keyPEM, err := sso.GenerateSPCredentials(settings.SPEntityID)
			if err != nil {
				return nil, "", err
			}
			settings.SPCertPEM = cert
			secret = keyPEM
		}
		encoded, err := json.Marshal(settings)
		return encoded, secret, err
	default:
		return raw, secret, nil
	}
}

func putProviderSecret(ctx context.Context, store secrets.Store, ref, plaintext string) error {
	if store == nil {
		return errors.New("DATA_STORE_SECRET_KEY is required to store SSO secrets")
	}
	return store.Put(ctx, ref, []byte(plaintext))
}

func scanAuthProvider(row pgx.Row) (authProviderResponse, error) {
	var item authProviderResponse
	var config []byte
	err := row.Scan(&item.ID, &item.Slug, &item.Type, &item.DisplayName, &item.Enabled, &item.AutoProvision, &item.DefaultRole, &item.LinkByEmail, &item.HasSecret, &config)
	if len(config) == 0 {
		config = []byte(`{}`)
	}
	item.Config = config
	return item, err
}

func writeProviderStoreError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "provider slug already exists"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "save provider failed"})
}

func boolOr(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func roleOr(role string) string {
	if role == "" {
		return "viewer"
	}
	return role
}
