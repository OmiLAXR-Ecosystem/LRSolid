package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/config"
)

type restUser struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"displayName,omitempty"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	DisabledAt  *string `json:"disabledAt,omitempty"`
}

type restListResponse struct {
	Items  []restUser `json:"items"`
	Total  int64      `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

func RestListUsers(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, err := getOrgID(ctx, db, cfg.DefaultOrg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusOK, restListResponse{Items: []restUser{}, Total: 0, Limit: 0, Offset: 0})
			return
		}

		q := r.URL.Query()
		limit := parseLimit(q.Get("limit"), 25, 200)
		offset := parseOffset(q.Get("offset"))
		role := strings.TrimSpace(q.Get("role"))
		search := strings.TrimSpace(q.Get("q"))

		args := []any{orgID}
		conds := []string{"om.org_id = $1"}
		argn := 2
		if role != "" && role != "All" {
			conds = append(conds, "om.role = $"+strconv.Itoa(argn))
			args = append(args, role)
			argn++
		}
		if search != "" {
			conds = append(conds, "(u.email ILIKE $"+strconv.Itoa(argn)+" OR u.display_name ILIKE $"+strconv.Itoa(argn)+")")
			args = append(args, "%"+search+"%")
			argn++
		}

		where := strings.Join(conds, " AND ")

		var total int64
		countQ := "SELECT COUNT(*) FROM users u JOIN org_memberships om ON om.user_id = u.id WHERE " + where
		if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "count users failed"})
			return
		}

		args = append(args, limit, offset)
		listQ := `
			SELECT u.id, u.display_name, u.email, om.role, u.disabled_at
			FROM users u
			JOIN org_memberships om ON om.user_id = u.id
			WHERE ` + where + `
			ORDER BY u.created_at DESC
			LIMIT $` + strconv.Itoa(argn) + ` OFFSET $` + strconv.Itoa(argn+1)

		rows, err := db.Query(ctx, listQ, args...)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list users failed"})
			return
		}
		defer rows.Close()

		items := make([]restUser, 0, limit)
		for rows.Next() {
			var u restUser
			var disabledAt *string
			if err := rows.Scan(&u.ID, &u.DisplayName, &u.Email, &u.Role, &disabledAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan users failed"})
				return
			}
			u.DisabledAt = disabledAt
			items = append(items, u)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read users failed"})
			return
		}

		writeJSON(w, http.StatusOK, restListResponse{Items: items, Total: total, Limit: limit, Offset: offset})
	}
}

func RestCountUsers(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, err := getOrgID(ctx, db, cfg.DefaultOrg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusOK, map[string]any{"count": 0})
			return
		}

		var total int64
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users u JOIN org_memberships om ON om.user_id = u.id WHERE om.org_id=$1`, orgID).Scan(&total); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "count users failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"count": total})
	}
}

func restNotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "not implemented"})
	}
}

var _ = pgx.ErrNoRows
