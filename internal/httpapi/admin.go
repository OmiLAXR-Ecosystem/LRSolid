package httpapi

import (
	"context"
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
	"github.com/SGoerzen/LRSolid/internal/lrs"
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

// ---- Dashboard ----

type dashboardResponse struct {
	KPIs struct {
		TotalStatements int64 `json:"totalStatements"`
		ActiveUsers     int64 `json:"activeUsers"`
		StatementsToday int64 `json:"statementsToday"`
		ErrorLogs       int64 `json:"errorLogs"`
	} `json:"kpis"`
	RecentStatements []dashboardStatement `json:"recentStatements"`
}

type dashboardStatement struct {
	ID        string     `json:"id"`
	ActorMBox *string    `json:"actorMbox,omitempty"`
	VerbID    *string    `json:"verbId,omitempty"`
	ObjectID  *string    `json:"objectId,omitempty"`
	StoredAt  time.Time  `json:"storedAt"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

func AdminDashboard(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID(cfg.DefaultTenant)
		ctx := r.Context()

		var total int64
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM statements WHERE tenant_id=$1`, tenant).Scan(&total); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query total statements failed"})
			return
		}

		now := time.Now().UTC()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)
		var today int64
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*) FROM statements
			WHERE tenant_id=$1 AND stored_at >= $2 AND stored_at < $3
		`, tenant, startOfDay, endOfDay).Scan(&today); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query statements today failed"})
			return
		}

		lookback := now.Add(-30 * 24 * time.Hour)
		var active int64
		if err := db.QueryRow(ctx, `
			SELECT COUNT(DISTINCT actor_mbox)
			FROM statements
			WHERE tenant_id=$1 AND actor_mbox IS NOT NULL AND stored_at >= $2
		`, tenant, lookback).Scan(&active); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query active users failed"})
			return
		}

		rows, err := db.Query(ctx, `
			SELECT id, actor_mbox, verb_id, object_id, stored_at, (statement->>'timestamp')::timestamptz
			FROM statements
			WHERE tenant_id=$1
			ORDER BY stored_at DESC, id DESC
			LIMIT 5
		`, tenant)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query recent statements failed"})
			return
		}
		defer rows.Close()

		recent := make([]dashboardStatement, 0, 5)
		for rows.Next() {
			var st dashboardStatement
			var ts *time.Time
			if err := rows.Scan(&st.ID, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.StoredAt, &ts); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan recent statements failed"})
				return
			}
			st.Timestamp = ts
			recent = append(recent, st)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read recent statements failed"})
			return
		}

		var resp dashboardResponse
		resp.KPIs.TotalStatements = total
		resp.KPIs.ActiveUsers = active
		resp.KPIs.StatementsToday = today
		resp.KPIs.ErrorLogs = 0
		resp.RecentStatements = recent

		writeJSON(w, http.StatusOK, resp)
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
}

func AdminCreateUser(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		if req.Email == "" || req.Password == "" || req.Role == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "email, password, role are required"})
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
