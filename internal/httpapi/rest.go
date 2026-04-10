package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

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
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	LRSID     *string         `json:"lrsId,omitempty"`
	LRSName   *string         `json:"lrsName,omitempty"`
	Name      string          `json:"name"`
	Note      *string         `json:"note,omitempty"`
	KeyID     string          `json:"keyId"`
	Scopes    json.RawMessage `json:"scopes"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt *time.Time      `json:"expiresAt,omitempty"`
	RevokedAt *time.Time      `json:"revokedAt,omitempty"`
	UserIDs   []string        `json:"userIds"`
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
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	LRSID     *string         `json:"lrsId,omitempty"`
	LRSName   *string         `json:"lrsName,omitempty"`
	Name      string          `json:"name"`
	Note      *string         `json:"note,omitempty"`
	KeyID     string          `json:"keyId"`
	Secret    string          `json:"secret"`
	Scopes    json.RawMessage `json:"scopes"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt *time.Time      `json:"expiresAt,omitempty"`
	UserIDs   []string        `json:"userIds"`
}

type restLRS struct {
	ID            string `json:"id"`
	ProjectID     string `json:"projectId"`
	Name          string `json:"name"`
	DataStoreID   string `json:"dataStoreId"`
	DataStoreName string `json:"dataStoreName"`
}

type restLRSListResponse struct {
	Items []restLRS `json:"items"`
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

func RestListLRS(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(r.Context(), `
				SELECT l.id, l.project_id, l.name, ds.id, ds.name
				FROM lrs l
				JOIN data_stores ds ON ds.id = l.data_store_id
				JOIN projects p ON p.id = l.project_id
				JOIN organisations o ON o.id = p.org_id
				WHERE o.slug = $1
				ORDER BY l.name ASC, l.id ASC
			`, cfg.DefaultOrg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list learning record stores failed"})
			return
		}
		defer rows.Close()

		items := make([]restLRS, 0, 16)
		for rows.Next() {
			var item restLRS
			if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.DataStoreID, &item.DataStoreName); err != nil {
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
		q := r.URL.Query()
		limit := parseLimit(q.Get("limit"), 25, 200)
		offset := parseOffset(q.Get("offset"))
		search := strings.TrimSpace(q.Get("q"))

		args := []any{cfg.DefaultOrg}
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

		projectID, lrsName, err := resolveProjectForLRS(ctx, tx, cfg.DefaultOrg, lrsID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		userIDs, err := validateOrgUsers(ctx, tx, cfg.DefaultOrg, req.UserIDs)
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
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "client id required"})
			return
		}
		ct, err := db.Exec(r.Context(), `
			DELETE FROM clients c
			USING projects p, organisations o
			WHERE c.id = $1
			  AND c.project_id = p.id
			  AND p.org_id = o.id
			  AND o.slug = $2
		`, id, cfg.DefaultOrg)
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

var _ = pgx.ErrNoRows
