package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
)

type clientAuthContextKey string

const xapiTenantKey clientAuthContextKey = "xapi_tenant"

func tenantFromRequest(ctx context.Context, cfg config.Config) string {
	if tenant, ok := ctx.Value(xapiTenantKey).(string); ok && tenant != "" {
		return tenant
	}
	return cfg.DefaultTenant
}

func xapiScopeAllowed(scopes []byte, scope string) bool {
	if len(scopes) == 0 {
		return false
	}
	var m map[string]bool
	if err := json.Unmarshal(scopes, &m); err != nil {
		return false
	}
	return m[scope]
}

func requiredStatementScope(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodPost:
		return "statements:write"
	case http.MethodGet:
		return "statements:read"
	default:
		return ""
	}
}

// XAPIClientAuth authenticates xAPI requests with HTTP Basic auth (key_id:secret).
// If authenticated, it injects the tenant to use for statement access into request context.
func XAPIClientAuth(cfg config.Config, db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyID, secret, ok := r.BasicAuth()
			if !ok || keyID == "" || secret == "" {
				// Backward-compatible fallback for existing web UI/session flows.
				cookie, err := r.Cookie(auth.SessionCookie)
				if err == nil && cookie != nil && cookie.Value != "" {
					if _, err := auth.ValidateToken(cfg.JWTSecret, cookie.Value); err == nil {
						next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), xapiTenantKey, cfg.DefaultTenant)))
						return
					}
				}
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "basic auth required"})
				return
			}

			requiredScope := requiredStatementScope(r.Method)
			if requiredScope == "" {
				writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
				return
			}

			var secretHash string
			var lrsID *string
			var scopesJSON []byte
			var expiresAt *time.Time
			err := db.QueryRow(r.Context(), `
				SELECT secret_hash, lrs_id, scopes::text, expires_at
				FROM clients
				WHERE key_id = $1
				  AND revoked_at IS NULL
				LIMIT 1
			`, keyID).Scan(&secretHash, &lrsID, &scopesJSON, &expiresAt)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid client credentials"})
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(secret)); err != nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid client credentials"})
				return
			}

			if !xapiScopeAllowed(scopesJSON, requiredScope) {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "insufficient scope"})
				return
			}
			if expiresAt != nil && time.Now().UTC().After(*expiresAt) {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "client credentials expired"})
				return
			}

			tenant := cfg.DefaultTenant
			if lrsID != nil && *lrsID != "" {
				tenant = *lrsID
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), xapiTenantKey, tenant)))
		})
	}
}
