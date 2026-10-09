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
const xapiClientIDKey clientAuthContextKey = "xapi_client_id"
const xapiDataStoreIDKey clientAuthContextKey = "xapi_data_store_id"
const xapiKeyIDKey clientAuthContextKey = "xapi_key_id"
const xapiMineOnlyKey clientAuthContextKey = "xapi_mine_only"
const xapiDefineKey clientAuthContextKey = "xapi_define"

func tenantFromRequest(ctx context.Context, cfg config.Config) string {
	if tenant, ok := ctx.Value(xapiTenantKey).(string); ok && tenant != "" {
		return tenant
	}
	return cfg.DefaultTenant
}

func dataStoreIDFromRequest(ctx context.Context) string {
	if id, ok := ctx.Value(xapiDataStoreIDKey).(string); ok {
		return id
	}
	return ""
}

func clientIDFromRequest(ctx context.Context) *string {
	if clientID, ok := ctx.Value(xapiClientIDKey).(string); ok && clientID != "" {
		return &clientID
	}
	return nil
}

func keyIDFromRequest(ctx context.Context) string {
	if keyID, ok := ctx.Value(xapiKeyIDKey).(string); ok {
		return keyID
	}
	return ""
}

func mineOnlyFromRequest(ctx context.Context) bool {
	mine, _ := ctx.Value(xapiMineOnlyKey).(bool)
	return mine
}

func defineFromRequest(ctx context.Context) bool {
	ok, _ := ctx.Value(xapiDefineKey).(bool)
	return ok
}

func xapiResource(path string) string {
	p := strings.Trim(strings.TrimPrefix(path, "/xapi"), "/")
	switch {
	case p == "statements" || strings.HasPrefix(p, "statements/"):
		return "statements"
	case p == "activities/state":
		return "state"
	case p == "activities/profile" || p == "agents/profile":
		return "profile"
	case p == "activities":
		return "activities"
	case p == "agents":
		return "agents"
	default:
		return ""
	}
}

func scopeMap(scopes []byte) (map[string]bool, bool) {
	if len(scopes) == 0 {
		return nil, false
	}
	var m map[string]bool
	if err := json.Unmarshal(scopes, &m); err != nil {
		return nil, false
	}
	return m, true
}

func hasScope(m map[string]bool, names ...string) bool {
	for _, name := range names {
		if m[name] {
			return true
		}
	}
	return false
}

// xapiPermitted reports whether the scope map allows the method on a resource.
// mineOnly is set when a statements read is limited to statements this client stored.
func xapiPermitted(scopes []byte, method, path string) (bool, bool) {
	m, ok := scopeMap(scopes)
	if !ok {
		return false, false
	}
	if m["all"] {
		return true, false
	}
	read := method == http.MethodGet || method == http.MethodHead
	switch xapiResource(path) {
	case "statements":
		if read {
			if hasScope(m, "all/read", "statements/read", "statements:read") {
				return true, false
			}
			if hasScope(m, "statements/read/mine", "statements/write", "statements:write") {
				return true, true
			}
			return false, false
		}
		return hasScope(m, "statements/write", "statements:write"), false
	case "state":
		if read {
			return hasScope(m, "state", "all/read"), false
		}
		return hasScope(m, "state"), false
	case "profile":
		if read {
			return hasScope(m, "profile", "all/read"), false
		}
		return hasScope(m, "profile"), false
	case "activities", "agents":
		if !read {
			return false, false
		}
		return hasScope(m, "all/read", "statements/read", "statements:read", "statements/read/mine", "statements/write", "statements:write", "state", "profile", "define"), false
	default:
		return false, false
	}
}

func xapiCanDefine(scopes []byte) bool {
	m, ok := scopeMap(scopes)
	if !ok {
		return false
	}
	return hasScope(m, "all", "define")
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
						ctx := context.WithValue(r.Context(), xapiTenantKey, cfg.DefaultTenant)
						ctx = context.WithValue(ctx, xapiKeyIDKey, "session")
						ctx = context.WithValue(ctx, xapiDefineKey, true)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "basic auth required"})
				return
			}

			var clientID string
			var secretHash string
			var lrsID *string
			var dataStoreID *string
			var scopesJSON []byte
			var expiresAt *time.Time
			err := db.QueryRow(r.Context(), `
				SELECT c.id::text, c.secret_hash, c.lrs_id, l.data_store_id::text, c.scopes::text, c.expires_at
				FROM clients c
				LEFT JOIN lrs l ON l.id = c.lrs_id
				WHERE c.key_id = $1
				  AND c.revoked_at IS NULL
				LIMIT 1
			`, keyID).Scan(&clientID, &secretHash, &lrsID, &dataStoreID, &scopesJSON, &expiresAt)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid client credentials"})
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(secret)); err != nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid client credentials"})
				return
			}

			allowed, mineOnly := xapiPermitted(scopesJSON, r.Method, r.URL.Path)
			if !allowed {
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

			ctx := context.WithValue(r.Context(), xapiTenantKey, tenant)
			ctx = context.WithValue(ctx, xapiClientIDKey, clientID)
			ctx = context.WithValue(ctx, xapiKeyIDKey, keyID)
			ctx = context.WithValue(ctx, xapiMineOnlyKey, mineOnly)
			ctx = context.WithValue(ctx, xapiDefineKey, xapiCanDefine(scopesJSON))
			if dataStoreID != nil && *dataStoreID != "" {
				ctx = context.WithValue(ctx, xapiDataStoreIDKey, *dataStoreID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
