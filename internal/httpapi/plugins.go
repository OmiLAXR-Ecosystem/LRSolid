package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type pluginInfo struct {
	ID     string                `json:"id"`
	Schema []plugins.ConfigField `json:"schema"`
}

// GET /api/v1/{org}/plugins
// Returns all plugins currently loaded in the registry (builtin + filesystem).
func RestListPlugins(_ config.Config, db *pgxpool.Pool, registry *plugins.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := strings.TrimSpace(chi.URLParam(r, "org"))
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		items := make([]pluginInfo, 0, len(registry.All()))
		for _, p := range registry.All() {
			items = append(items, pluginInfo{ID: p.ID(), Schema: p.Schema()})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
