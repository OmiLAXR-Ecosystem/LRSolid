package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/SGoerzen/LRSolid/internal/auth"
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

type restClient struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"projectId"`
	LRSID          *string         `json:"lrsId,omitempty"`
	LRSName        *string         `json:"lrsName,omitempty"`
	Name           string          `json:"name"`
	StatementCount int64           `json:"statementCount"`
	Note           *string         `json:"note,omitempty"`
	KeyID          string          `json:"keyId"`
	Scopes         json.RawMessage `json:"scopes"`
	CreatedAt      time.Time       `json:"createdAt"`
	ExpiresAt      *time.Time      `json:"expiresAt,omitempty"`
	RevokedAt      *time.Time      `json:"revokedAt,omitempty"`
	UserIDs        []string        `json:"userIds"`
}

type restClientListResponse struct {
	Items  []restClient `json:"items"`
	Total  int64        `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type createClientRequest struct {
	Name    string          `json:"name"`
	LRSID   *string         `json:"lrsId,omitempty"`
	Note    *string         `json:"note,omitempty"`
	Scopes  json.RawMessage `json:"scopes,omitempty"`
	Expires *string         `json:"expiresAt,omitempty"`
	UserIDs []string        `json:"userIds,omitempty"`
}

type createClientResponse struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"projectId"`
	LRSID          *string         `json:"lrsId,omitempty"`
	LRSName        *string         `json:"lrsName,omitempty"`
	Name           string          `json:"name"`
	StatementCount int64           `json:"statementCount"`
	Note           *string         `json:"note,omitempty"`
	KeyID          string          `json:"keyId"`
	Secret         string          `json:"secret"`
	Scopes         json.RawMessage `json:"scopes"`
	CreatedAt      time.Time       `json:"createdAt"`
	ExpiresAt      *time.Time      `json:"expiresAt,omitempty"`
	UserIDs        []string        `json:"userIds"`
}

type restLRS struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	Name           string    `json:"name"`
	DataStoreID    string    `json:"dataStoreId"`
	DataStoreName  string    `json:"dataStoreName"`
	StatementCount int64     `json:"statementCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

type restLRSListResponse struct {
	Items []restLRS `json:"items"`
}

type createLRSRequest struct {
	Name        string `json:"name"`
	ProjectID   string `json:"projectId"`
	DataStoreID string `json:"dataStoreId"`
}

type restProject struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	StatementCount int64     `json:"statementCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

type restProjectListResponse struct {
	Items []restProject `json:"items"`
}

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type restDataStore struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DatabaseType string `json:"databaseType"`
	Host         string `json:"host"`
	IsDefault    bool   `json:"isDefault"`
}

type restDataStoreListResponse struct {
	Items []restDataStore `json:"items"`
}

type restOrg struct {
	ID             string `json:"id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	StatementCount int64  `json:"statementCount"`
}

type restOrgListResponse struct {
	Items []restOrg `json:"items"`
}

// validateOrgAccess checks that the current user may access the given org slug.
// Admins and owners bypass the membership check.
func validateOrgAccess(ctx context.Context, db *pgxpool.Pool, claims auth.Claims, orgSlug string) error {
	if claims.Role == "admin" || claims.Role == "owner" {
		return nil
	}
	var exists bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM org_memberships om
			JOIN organisations o ON o.id = om.org_id
			WHERE om.user_id = $1::uuid AND o.slug = $2
		)
	`, claims.UserID, orgSlug).Scan(&exists); err != nil {
		return fmt.Errorf("check org access failed")
	}
	if !exists {
		return fmt.Errorf("access denied")
	}
	return nil
}

// orgFromRequest extracts the {org} URL param, falling back to cfg.DefaultOrg.
func orgFromRequest(r *http.Request, cfg config.Config) string {
	if s := chi.URLParam(r, "org"); s != "" {
		return s
	}
	return cfg.DefaultOrg
}

func RestListOrgs(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}

		var (
			rows pgx.Rows
			err  error
		)
		if claims.Role == "admin" || claims.Role == "owner" {
			rows, err = db.Query(ctx, `SELECT id, slug, name, statement_count FROM organisations ORDER BY name ASC`)
		} else {
			rows, err = db.Query(ctx, `
				SELECT o.id, o.slug, o.name, o.statement_count
				FROM organisations o
				JOIN org_memberships om ON om.org_id = o.id
				WHERE om.user_id = $1::uuid
				ORDER BY o.name ASC
			`, claims.UserID)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list orgs failed"})
			return
		}
		defer rows.Close()

		items := make([]restOrg, 0)
		for rows.Next() {
			var o restOrg
			if err := rows.Scan(&o.ID, &o.Slug, &o.Name, &o.StatementCount); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan orgs failed"})
				return
			}
			items = append(items, o)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read orgs failed"})
			return
		}

		writeJSON(w, http.StatusOK, restOrgListResponse{Items: items})
	}
}

func RestListUsers(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		orgID, err := getOrgID(ctx, db, orgSlug)
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
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		orgID, err := getOrgID(ctx, db, orgSlug)
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

func RestListLRS(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		rows, err := db.Query(ctx, `
				SELECT l.id, l.project_id, l.name, ds.id, ds.name, l.statement_count, l.created_at
				FROM lrs l
				JOIN data_stores ds ON ds.id = l.data_store_id
				JOIN projects p ON p.id = l.project_id
				JOIN organisations o ON o.id = p.org_id
				WHERE o.slug = $1
				ORDER BY l.name ASC, l.id ASC
			`, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list learning record stores failed"})
			return
		}
		defer rows.Close()

		items := make([]restLRS, 0, 16)
		for rows.Next() {
			var item restLRS
			if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.DataStoreID, &item.DataStoreName, &item.StatementCount, &item.CreatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan learning record stores failed"})
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read learning record stores failed"})
			return
		}

		writeJSON(w, http.StatusOK, restLRSListResponse{Items: items})
	}
}

func RestListClients(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		q := r.URL.Query()
		limit := parseLimit(q.Get("limit"), 25, 200)
		offset := parseOffset(q.Get("offset"))
		search := strings.TrimSpace(q.Get("q"))

		args := []any{orgSlug}
		where := "o.slug = $1"
		if search != "" {
			args = append(args, "%"+search+"%")
			where += " AND (c.name ILIKE $2 OR c.key_id ILIKE $2 OR c.note ILIKE $2)"
		}

		var total int64
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM clients c
			JOIN projects p ON p.id = c.project_id
			JOIN organisations o ON o.id = p.org_id
			WHERE `+where+`
		`, args...).Scan(&total); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "count clients failed"})
			return
		}

		args = append(args, limit, offset)
		rows, err := db.Query(ctx, `
			SELECT
				c.id,
				c.project_id,
				c.lrs_id,
				l.name AS lrs_name,
				c.name,
				c.statement_count,
				c.note,
				c.key_id,
				c.scopes::text,
				c.created_at,
				c.expires_at,
				c.revoked_at,
				COALESCE(
					(
						SELECT json_agg(cau.user_id::text)
						FROM client_allowed_users cau
						WHERE cau.client_id = c.id
					)::text,
					'[]'
				) AS user_ids
			FROM clients c
			LEFT JOIN lrs l ON l.id = c.lrs_id
			JOIN projects p ON p.id = c.project_id
			JOIN organisations o ON o.id = p.org_id
			WHERE `+where+`
			ORDER BY c.created_at DESC
			LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args))+`
		`, args...)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list clients failed"})
			return
		}
		defer rows.Close()

		items := make([]restClient, 0, limit)
		for rows.Next() {
			var c restClient
			var scopesText string
			var userIDsText string
			if err := rows.Scan(
				&c.ID,
				&c.ProjectID,
				&c.LRSID,
				&c.LRSName,
				&c.Name,
				&c.StatementCount,
				&c.Note,
				&c.KeyID,
				&scopesText,
				&c.CreatedAt,
				&c.ExpiresAt,
				&c.RevokedAt,
				&userIDsText,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan clients failed"})
				return
			}
			c.Scopes = json.RawMessage(scopesText)
			if err := json.Unmarshal([]byte(userIDsText), &c.UserIDs); err != nil {
				c.UserIDs = []string{}
			}
			items = append(items, c)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read clients failed"})
			return
		}

		writeJSON(w, http.StatusOK, restClientListResponse{
			Items:  items,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

func RestCreateClient(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		var req createClientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
			return
		}
		if req.LRSID == nil || strings.TrimSpace(*req.LRSID) == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "lrsId is required"})
			return
		}
		lrsID := strings.TrimSpace(*req.LRSID)

		scopes := req.Scopes
		if len(scopes) == 0 {
			scopes = json.RawMessage(`{"statements:read":true,"statements:write":true}`)
		} else {
			var v map[string]bool
			if err := json.Unmarshal(scopes, &v); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "scopes must be an object of booleans"})
				return
			}
		}
		var expiresAt *time.Time
		if req.Expires != nil && strings.TrimSpace(*req.Expires) != "" {
			parsed, err := parseISODateOrTimestamp(strings.TrimSpace(*req.Expires))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "expiresAt must be RFC3339 or YYYY-MM-DD"})
				return
			}
			expiresAt = &parsed
		}
		note := normalizeOptionalString(req.Note)

		tx, err := db.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "begin transaction failed"})
			return
		}
		defer tx.Rollback(ctx)

		projectID, lrsName, err := resolveProjectForLRS(ctx, tx, orgSlug, lrsID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		userIDs, err := validateOrgUsers(ctx, tx, orgSlug, req.UserIDs)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		keyID, secret, err := generateClientCredentials()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "generate credentials failed"})
			return
		}
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "hash secret failed"})
			return
		}

		var out createClientResponse
		out.ProjectID = projectID
		out.LRSID = &lrsID
		out.LRSName = &lrsName
		out.Name = req.Name
		out.Note = note
		out.KeyID = keyID
		out.Secret = secret
		out.Scopes = scopes
		out.StatementCount = 0
		out.ExpiresAt = expiresAt
		out.UserIDs = userIDs
		err = tx.QueryRow(ctx, `
			INSERT INTO clients (project_id, lrs_id, name, note, key_id, secret_hash, scopes, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
			RETURNING id, created_at
		`, projectID, lrsID, req.Name, note, keyID, string(hashBytes), string(scopes), expiresAt).Scan(&out.ID, &out.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create client failed"})
			return
		}
		for _, userID := range userIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO client_allowed_users (client_id, user_id)
				VALUES ($1, $2)
			`, out.ID, userID); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "assign client users failed"})
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "commit failed"})
			return
		}

		writeJSON(w, http.StatusCreated, out)
	}
}

func RestDeleteClient(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "client id required"})
			return
		}
		ct, err := db.Exec(ctx, `
			DELETE FROM clients c
			USING projects p, organisations o
			WHERE c.id = $1
			  AND c.project_id = p.id
			  AND p.org_id = o.id
			  AND o.slug = $2
		`, id, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete client failed"})
			return
		}
		if ct.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "client not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}

func parseISODateOrTimestamp(s string) (time.Time, error) {
	if len(s) == 10 && strings.Count(s, "-") == 2 {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func normalizeOptionalString(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func resolveProjectForLRS(ctx context.Context, tx pgx.Tx, orgSlug, lrsID string) (projectID, lrsName string, err error) {
	err = tx.QueryRow(ctx, `
		SELECT l.project_id, l.name
		FROM lrs l
		JOIN projects p ON p.id = l.project_id
		JOIN organisations o ON o.id = p.org_id
		WHERE l.id = $1
		  AND o.slug = $2
		LIMIT 1
	`, lrsID, orgSlug).Scan(&projectID, &lrsName)
	if err == pgx.ErrNoRows {
		return "", "", errors.New("data store not found in organisation")
	}
	if err != nil {
		return "", "", err
	}
	return projectID, lrsName, nil
}

func validateOrgUsers(ctx context.Context, tx pgx.Tx, orgSlug string, userIDs []string) ([]string, error) {
	out := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, raw := range userIDs {
		userID := strings.TrimSpace(raw)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		var exists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM org_memberships om
				JOIN organisations o ON o.id = om.org_id
				WHERE om.user_id = $1::uuid
				  AND o.slug = $2
			)
		`, userID, orgSlug).Scan(&exists)
		if err != nil {
			return nil, errors.New("invalid user id in userIds")
		}
		if !exists {
			return nil, errors.New("one or more selected users are not in the organisation")
		}
		out = append(out, userID)
	}
	return out, nil
}

func generateClientCredentials() (string, string, error) {
	keyBytes := make([]byte, 12)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", err
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}

	keyID := "ck_" + hex.EncodeToString(keyBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	return keyID, secret, nil
}

func restNotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "not implemented"})
	}
}

func RestListProjects(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		rows, err := db.Query(ctx, `
			SELECT p.id, p.slug, p.name, p.description, p.statement_count, p.created_at
			FROM projects p
			JOIN organisations o ON o.id = p.org_id
			WHERE o.slug = $1
			ORDER BY p.name ASC
		`, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list projects failed"})
			return
		}
		defer rows.Close()

		items := make([]restProject, 0, 8)
		for rows.Next() {
			var p restProject
			if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.StatementCount, &p.CreatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan projects failed"})
				return
			}
			items = append(items, p)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read projects failed"})
			return
		}

		writeJSON(w, http.StatusOK, restProjectListResponse{Items: items})
	}
}

func RestCreateProject(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		var req createProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
			return
		}
		description := normalizeOptionalString(req.Description)

		slugBytes := make([]byte, 4)
		if _, err := rand.Read(slugBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "generate slug failed"})
			return
		}
		generatedSlug := hex.EncodeToString(slugBytes)

		var orgID string
		if err := db.QueryRow(ctx, `SELECT id FROM organisations WHERE slug = $1`, orgSlug).Scan(&orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "organisation not found"})
			return
		}

		var out restProject
		err := db.QueryRow(ctx, `
			INSERT INTO projects (org_id, slug, name, description)
			VALUES ($1, $2, $3, $4)
			RETURNING id, slug, name, description, statement_count, created_at
		`, orgID, generatedSlug, req.Name, description).Scan(&out.ID, &out.Slug, &out.Name, &out.Description, &out.StatementCount, &out.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create project failed"})
			return
		}

		writeJSON(w, http.StatusCreated, out)
	}
}

func RestDeleteProject(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "project id required"})
			return
		}

		ct, err := db.Exec(ctx, `
			DELETE FROM projects p
			USING organisations o
			WHERE p.id = $1
			  AND p.org_id = o.id
			  AND o.slug = $2
		`, id, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete project failed"})
			return
		}
		if ct.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "project not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}

func RestListDataStores(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(r.Context(), `
			SELECT id, name, database_type, host, is_default
			FROM data_stores
			WHERE is_active = true
			ORDER BY is_default DESC, name ASC
		`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list data stores failed"})
			return
		}
		defer rows.Close()

		items := make([]restDataStore, 0, 8)
		for rows.Next() {
			var ds restDataStore
			if err := rows.Scan(&ds.ID, &ds.Name, &ds.DatabaseType, &ds.Host, &ds.IsDefault); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan data stores failed"})
				return
			}
			items = append(items, ds)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read data stores failed"})
			return
		}

		writeJSON(w, http.StatusOK, restDataStoreListResponse{Items: items})
	}
}

func RestCreateLRS(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		var req createLRSRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
			return
		}
		req.DataStoreID = strings.TrimSpace(req.DataStoreID)
		if req.DataStoreID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "dataStoreId is required"})
			return
		}
		req.ProjectID = strings.TrimSpace(req.ProjectID)
		if req.ProjectID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "projectId is required"})
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "begin transaction failed"})
			return
		}
		defer tx.Rollback(ctx)

		// Verify the project belongs to this org
		var projectID string
		err = tx.QueryRow(ctx, `
			SELECT p.id FROM projects p
			JOIN organisations o ON o.id = p.org_id
			WHERE p.id = $1 AND o.slug = $2
		`, req.ProjectID, orgSlug).Scan(&projectID)
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "project not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "resolve project failed"})
			return
		}

		// Verify data store exists
		var dsName string
		err = tx.QueryRow(ctx, `
			SELECT name FROM data_stores WHERE id = $1 AND is_active = true
		`, req.DataStoreID).Scan(&dsName)
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "data store not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "verify data store failed"})
			return
		}

		var out restLRS
		out.ProjectID = projectID
		out.Name = req.Name
		out.DataStoreID = req.DataStoreID
		out.DataStoreName = dsName
		out.StatementCount = 0

		err = tx.QueryRow(ctx, `
			INSERT INTO lrs (id, project_id, name, data_store_id)
			VALUES (gen_random_uuid()::text, $1, $2, $3)
			RETURNING id, created_at
		`, projectID, req.Name, req.DataStoreID).Scan(&out.ID, &out.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create learning record store failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "commit failed"})
			return
		}

		writeJSON(w, http.StatusCreated, out)
	}
}

func RestDeleteLRS(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "lrs id required"})
			return
		}

		ct, err := db.Exec(ctx, `
			DELETE FROM lrs l
			USING projects p, organisations o
			WHERE l.id = $1
			  AND l.project_id = p.id
			  AND p.org_id = o.id
			  AND o.slug = $2
		`, id, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete learning record store failed"})
			return
		}
		if ct.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "learning record store not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}

var _ = pgx.ErrNoRows
