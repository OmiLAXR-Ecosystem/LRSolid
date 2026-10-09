package httpapi

import (
	"context"
	"net/http"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

type xapiVersionContextKey string

const xapiVersionKey xapiVersionContextKey = "xapi_version"

func versionFromRequest(ctx context.Context) string {
	if v, ok := ctx.Value(xapiVersionKey).(string); ok && v != "" {
		return v
	}
	return xapi.Version103
}

// XAPIVersion requires X-Experience-API-Version and echoes the negotiated version.
func XAPIVersion(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get(xapi.HeaderVersion)
			negotiated, ok := xapi.Negotiate(raw)
			if !ok {
				if raw == "" && cfg.XAPILegacyCompat {
					negotiated = xapi.Version103
				} else {
					http.Error(w, "X-Experience-API-Version is missing or unsupported", http.StatusBadRequest)
					return
				}
			}
			w.Header().Set(xapi.HeaderVersion, negotiated)
			ctx := context.WithValue(r.Context(), xapiVersionKey, negotiated)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
