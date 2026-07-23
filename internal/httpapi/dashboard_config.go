package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

// GET /api/v1/{org}/dashboard/config?scope=user|org
//
// Returns the effective config for the caller. Precedence: user > org > default.
// Also returns the plugin schema map so the frontend can render config UI.
func RestGetDashboardConfig(_ config.Config, db *pgxpool.Pool, registry *plugins.Registry) http.HandlerFunc {
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

		var orgID string
		if err := db.QueryRow(ctx, `SELECT id FROM organisations WHERE slug=$1`, orgSlug).Scan(&orgID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query org failed"})
			return
		}

		// Try user config first, then org config, then default.
		entries := loadConfig(ctx, db, orgID, &claims.UserID)
		if entries == nil {
			entries = loadConfig(ctx, db, orgID, nil)
		}
		if entries == nil {
			entries = registry.DefaultEntries()
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"widgets": entries,
			"schema":  registry.SchemaMap(),
		})
	}
}

// PUT /api/v1/{org}/dashboard/config?scope=user|org
//
// Saves a dashboard config. scope=user saves for the calling user; scope=org
// saves the org default (requires owner/admin role).
func RestPutDashboardConfig(_ config.Config, db *pgxpool.Pool, registry *plugins.Registry) http.HandlerFunc {
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

		var orgID string
		if err := db.QueryRow(ctx, `SELECT id FROM organisations WHERE slug=$1`, orgSlug).Scan(&orgID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query org failed"})
			return
		}

		scope := r.URL.Query().Get("scope")
		if scope != "user" && scope != "org" {
			scope = "user"
		}

		var body struct {
			Widgets []plugins.WidgetEntry `json:"widgets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}

		widgetsJSON, err := json.Marshal(body.Widgets)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "marshal widgets failed"})
			return
		}

		var userID *string
		if scope == "user" {
			userID = &claims.UserID
		}

		_, err = db.Exec(ctx, `
			INSERT INTO dashboard_configs (org_id, user_id, scope, widgets, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (org_id, user_id) DO UPDATE
			  SET widgets = EXCLUDED.widgets,
			      scope = EXCLUDED.scope,
			      updated_at = now()
		`, orgID, userID, scope, widgetsJSON)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "save config failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"widgets": body.Widgets})
	}
}

// DELETE /api/v1/{org}/dashboard/config?scope=user
// Resets the calling user's config (falls back to org/default).
func RestDeleteDashboardConfig(_ config.Config, db *pgxpool.Pool) http.HandlerFunc {
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

		var orgID string
		if err := db.QueryRow(ctx, `SELECT id FROM organisations WHERE slug=$1`, orgSlug).Scan(&orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query org failed"})
			return
		}

		_, err := db.Exec(ctx, `
			DELETE FROM dashboard_configs WHERE org_id=$1 AND user_id=$2
		`, orgID, claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete config failed"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// loadConfig reads a dashboard config row. Pass nil userID for the org-level config.
// Returns nil if no config exists.
func loadConfig(ctx context.Context, db *pgxpool.Pool, orgID string, userID *string) []plugins.WidgetEntry {
	var raw []byte
	var err error
	if userID == nil {
		err = db.QueryRow(ctx, `SELECT widgets FROM dashboard_configs WHERE org_id=$1 AND user_id IS NULL`, orgID).Scan(&raw)
	} else {
		err = db.QueryRow(ctx, `SELECT widgets FROM dashboard_configs WHERE org_id=$1 AND user_id=$2`, orgID, *userID).Scan(&raw)
	}
	if err != nil {
		return nil
	}
	var entries []plugins.WidgetEntry
	if json.Unmarshal(raw, &entries) != nil {
		return nil
	}
	return entries
}
