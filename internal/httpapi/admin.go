package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/plugins"
	"github.com/SGoerzen/LRSolid/internal/store"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseLimit(qs string, def, max int) int {
	if qs == "" {
		return def
	}
	if n, err := strconv.Atoi(qs); err == nil && n > 0 {
		if n > max {
			return max
		}
		return n
	}
	return def
}

func parseOffset(qs string) int {
	if qs == "" {
		return 0
	}
	if n, err := strconv.Atoi(qs); err == nil && n >= 0 {
		return n
	}
	return 0
}

func parseTimeParam(s string) (*time.Time, bool, error) {
	if s == "" {
		return nil, false, nil
	}
	// Accept YYYY-MM-DD (treat as midnight UTC) or RFC3339/RFC3339Nano.
	if len(s) == 10 && strings.Count(s, "-") == 2 {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, false, err
		}
		utc := t.UTC()
		return &utc, true, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		utc := t.UTC()
		return &utc, false, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, false, err
	}
	utc := t.UTC()
	return &utc, false, nil
}

func getOrgID(ctx context.Context, db *pgxpool.Pool, slug string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT id FROM organisations WHERE slug=$1`, slug).Scan(&id)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return id, err
}

func requireAdminClaims(ctx context.Context) (auth.Claims, bool) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return auth.Claims{}, false
	}
	if claims.Role != "admin" && claims.Role != "owner" {
		return auth.Claims{}, false
	}
	return claims, true
}

func isValidOrgRole(role string) bool {
	switch role {
	case "owner", "admin", "member", "viewer":
		return true
	default:
		return false
	}
}

var orgSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func slugifyOrgName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(s))

	lastWasDash := false
	for _, r := range s {
		isAZ := r >= 'a' && r <= 'z'
		is09 := r >= '0' && r <= '9'
		if isAZ || is09 {
			b.WriteRune(r)
			lastWasDash = false
			continue
		}
		if b.Len() > 0 && !lastWasDash {
			b.WriteByte('-')
			lastWasDash = true
		}
	}

	base := strings.Trim(b.String(), "-")
	if base == "" {
		return "org"
	}
	return base
}

func randomShortIDHex(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generatedOrgSlug(name string) (string, error) {
	shortID, err := randomShortIDHex(3)
	if err != nil {
		return "", err
	}
	return slugifyOrgName(name) + "-" + shortID, nil
}

// ---- Dashboard ----

func AdminDashboard(_ config.Config, db *pgxpool.Pool, registry *plugins.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := strings.TrimSpace(chi.URLParam(r, "org"))
		if orgSlug == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "org is required"})
			return
		}
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
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query organisation failed"})
			return
		}

		// Precedence: user config → org config → default (all widgets).
		entries := loadConfig(ctx, db, orgID, &claims.UserID)
		if entries == nil {
			entries = loadConfig(ctx, db, orgID, nil)
		}
		if entries == nil {
			entries = registry.DefaultEntries()
		}

		widgets, err := registry.FetchAll(ctx, db, orgID, entries)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"widgets": widgets, "layout": entries})
	}
}

// ---- Statements ----

type adminStatement struct {
	ID        string          `json:"id"`
	ActorMBox *string         `json:"actorMbox,omitempty"`
	VerbID    *string         `json:"verbId,omitempty"`
	ObjectID  *string         `json:"objectId,omitempty"`
	Timestamp *time.Time      `json:"timestamp,omitempty"`
	StoredAt  time.Time       `json:"storedAt"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}

type adminStatementsResponse struct {
	Items      []adminStatement `json:"items"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

func AdminListStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID(cfg.DefaultTenant)
		q := r.URL.Query()

		limit := parseLimit(q.Get("limit"), 25, cfg.MaxPageLimit)
		cursor := q.Get("cursor")
		var cursorPtr *string
		if cursor != "" {
			cursorPtr = &cursor
		}

		since, _, err := parseTimeParam(q.Get("since"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid since"})
			return
		}
		until, untilDateOnly, err := parseTimeParam(q.Get("until"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid until"})
			return
		}
		if until != nil && untilDateOnly {
			end := until.Add(24*time.Hour - time.Nanosecond)
			until = &end
		}

		filter := store.StatementFilter{
			TenantID:    tenant,
			ActorMBox:   optString(q.Get("actor_mbox")),
			VerbID:      optString(q.Get("verb")),
			ObjectID:    optString(q.Get("activity")),
			StoredSince: since,
			StoredUntil: until,
			Limit:       limit,
			Ascending:   false,
			Cursor:      cursorPtr,
		}

		page, err := svc.QueryStatements(r.Context(), filter)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		items := make([]adminStatement, 0, len(page.Statements))
		for _, st := range page.Statements {
			items = append(items, adminStatement{
				ID:        string(st.ID),
				ActorMBox: st.ActorMBox,
				VerbID:    st.VerbID,
				ObjectID:  st.ObjectID,
				Timestamp: st.Timestamp,
				StoredAt:  st.StoredAt,
			})
		}

		writeJSON(w, http.StatusOK, adminStatementsResponse{
			Items:      items,
			NextCursor: page.NextCursor,
		})
	}
}

type adminStatementDetail struct {
	ID        string          `json:"id"`
	StoredAt  time.Time       `json:"storedAt"`
	Timestamp *time.Time      `json:"timestamp,omitempty"`
	ActorMBox *string         `json:"actorMbox,omitempty"`
	VerbID    *string         `json:"verbId,omitempty"`
	ObjectID  *string         `json:"objectId,omitempty"`
	Raw       json.RawMessage `json:"raw"`
}

func AdminGetStatement(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID(cfg.DefaultTenant)
		id := chi.URLParam(r, "id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id"})
			return
		}
		st, err := svc.GetStatementByID(r.Context(), tenant, store.StatementID(id))
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
			return
		}

		resp := adminStatementDetail{
			ID:        string(st.ID),
			StoredAt:  st.StoredAt,
			Timestamp: st.Timestamp,
			ActorMBox: st.ActorMBox,
			VerbID:    st.VerbID,
			ObjectID:  st.ObjectID,
			Raw:       json.RawMessage(st.RawJSON),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// ---- Users ----

type adminUser struct {
	ID           string     `json:"id"`
	Name         *string    `json:"name,omitempty"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	LastActivity *time.Time `json:"lastActivity,omitempty"`
	DisabledAt   *time.Time `json:"disabledAt,omitempty"`
}

type adminUsersResponse struct {
	Items  []adminUser `json:"items"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type createOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func AdminCreateOrg(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := requireAdminClaims(ctx); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}

		var req createOrgRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
			return
		}
		if req.Slug != "" && !orgSlugPattern.MatchString(req.Slug) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "slug must match [a-z0-9-] and cannot start/end with '-'"})
			return
		}

		insertWithSlug := func(slug string) (restOrg, error) {
			var org restOrg
			err := db.QueryRow(ctx, `
				INSERT INTO organisations (slug, name)
				VALUES ($1, $2)
				RETURNING id, slug, name, statement_count
			`, slug, req.Name).Scan(&org.ID, &org.Slug, &org.Name, &org.StatementCount)
			return org, err
		}

		if req.Slug != "" {
			org, err := insertWithSlug(req.Slug)
			if err != nil {
				if strings.Contains(err.Error(), "organisations_slug_key") {
					writeJSON(w, http.StatusConflict, errorResponse{Error: "organization slug already exists"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create organization failed"})
				return
			}
			writeJSON(w, http.StatusCreated, org)
			return
		}

		for i := 0; i < 5; i++ {
			autoSlug, err := generatedOrgSlug(req.Name)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create organization failed"})
				return
			}
			org, err := insertWithSlug(autoSlug)
			if err != nil {
				if strings.Contains(err.Error(), "organisations_slug_key") {
					continue
				}
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create organization failed"})
				return
			}
			writeJSON(w, http.StatusCreated, org)
			return
		}

		writeJSON(w, http.StatusConflict, errorResponse{Error: "organization slug already exists"})
		return
	}
}

func AdminListUsers(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		orgID, err := getOrgID(ctx, db, cfg.DefaultOrg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusOK, adminUsersResponse{Items: []adminUser{}, Total: 0, Limit: 0, Offset: 0})
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
			SELECT u.id, u.display_name, u.email, om.role, u.created_at, u.disabled_at
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

		items := make([]adminUser, 0, limit)
		for rows.Next() {
			var u adminUser
			var created time.Time
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &created, &u.DisabledAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan users failed"})
				return
			}
			u.LastActivity = &created
			items = append(items, u)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read users failed"})
			return
		}

		writeJSON(w, http.StatusOK, adminUsersResponse{
			Items:  items,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

type createUserRequest struct {
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
	Password    string  `json:"password"`
	Role        string  `json:"role"`
	OrgSlug     string  `json:"orgSlug"`
}

func AdminCreateUser(_ config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := requireAdminClaims(ctx); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}

		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		req.OrgSlug = strings.TrimSpace(req.OrgSlug)
		if req.Email == "" || req.Password == "" || req.Role == "" || req.OrgSlug == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "email, password, role, orgSlug are required"})
			return
		}
		if !isValidOrgRole(req.Role) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "role must be owner, admin, member, or viewer"})
			return
		}

		orgID, err := getOrgID(ctx, db, req.OrgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "org not found"})
			return
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "password hash failed"})
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "transaction failed"})
			return
		}
		defer tx.Rollback(ctx)

		var userID string
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, display_name, password_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (email) DO UPDATE
			  SET display_name = EXCLUDED.display_name,
			      password_hash = EXCLUDED.password_hash
			RETURNING id
		`, req.Email, req.DisplayName, string(hashBytes)).Scan(&userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create user failed"})
			return
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO org_memberships (org_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, orgID, userID, req.Role)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "assign role failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "commit failed"})
			return
		}

		writeJSON(w, http.StatusCreated, adminUser{
			ID:    userID,
			Name:  req.DisplayName,
			Email: req.Email,
			Role:  req.Role,
		})
	}
}

type assignUserToOrgRequest struct {
	Email   string `json:"email"`
	OrgSlug string `json:"orgSlug"`
	Role    string `json:"role"`
}

func AdminAssignUserToOrg(_ config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := requireAdminClaims(ctx); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}

		var req assignUserToOrgRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		req.OrgSlug = strings.TrimSpace(req.OrgSlug)
		if req.Email == "" || req.OrgSlug == "" || req.Role == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "email, orgSlug and role are required"})
			return
		}
		if !isValidOrgRole(req.Role) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "role must be owner, admin, member, or viewer"})
			return
		}

		orgID, err := getOrgID(ctx, db, req.OrgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organization not found"})
			return
		}

		var userID string
		if err := db.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, req.Email).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "user not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup user failed"})
			return
		}

		if _, err := db.Exec(ctx, `
			INSERT INTO org_memberships (org_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, orgID, userID, req.Role); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "assign user failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"userId":  userID,
			"email":   req.Email,
			"orgSlug": req.OrgSlug,
			"role":    req.Role,
		})
	}
}

type updateUserRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Role        *string `json:"role,omitempty"`
	Disabled    *bool   `json:"disabled,omitempty"`
}

func AdminUpdateUser(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "id")
		if userID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id"})
			return
		}

		var req updateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}

		orgID, err := getOrgID(ctx, db, cfg.DefaultOrg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "org not found"})
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "transaction failed"})
			return
		}
		defer tx.Rollback(ctx)

		if req.DisplayName != nil {
			if _, err := tx.Exec(ctx, `UPDATE users SET display_name=$1 WHERE id=$2`, req.DisplayName, userID); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "update display name failed"})
				return
			}
		}
		if req.Disabled != nil {
			if *req.Disabled {
				_, err = tx.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, userID)
			} else {
				_, err = tx.Exec(ctx, `UPDATE users SET disabled_at=NULL WHERE id=$1`, userID)
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "update disabled failed"})
				return
			}
		}
		if req.Role != nil {
			if _, err := tx.Exec(ctx, `
				INSERT INTO org_memberships (org_id, user_id, role)
				VALUES ($1, $2, $3)
				ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role
			`, orgID, userID, *req.Role); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "update role failed"})
				return
			}
		}

		var email string
		var name *string
		var role string
		var disabledAt *time.Time
		err = tx.QueryRow(ctx, `
			SELECT u.email, u.display_name, om.role, u.disabled_at
			FROM users u
			JOIN org_memberships om ON om.user_id = u.id
			WHERE u.id=$1 AND om.org_id=$2
		`, userID, orgID).Scan(&email, &name, &role, &disabledAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "fetch updated user failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "commit failed"})
			return
		}

		writeJSON(w, http.StatusOK, adminUser{
			ID:         userID,
			Name:       name,
			Email:      email,
			Role:       role,
			DisabledAt: disabledAt,
		})
	}
}
