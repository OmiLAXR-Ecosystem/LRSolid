package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/limits"
)

var limitSubjects = map[string]bool{
	limits.SubjectOrganisation: true,
	limits.SubjectProject:      true,
	limits.SubjectLRS:          true,
	limits.SubjectClient:       true,
}

var limitMetrics = map[string]bool{
	limits.MetricStatementsTotal:   true,
	limits.MetricStatementsPerHour: true,
	limits.MetricStatementsPerDay:  true,
	limits.MetricStorageBytes:      true,
}

type limitDTO struct {
	ID          string    `json:"id"`
	SubjectType string    `json:"subjectType"`
	SubjectID   string    `json:"subjectId"`
	Metric      string    `json:"metric"`
	MaxValue    int64     `json:"maxValue"`
	Enabled     bool      `json:"enabled"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type limitUsageDTO struct {
	StatementsTotal    int64 `json:"statementsTotal"`
	StatementsThisHour int64 `json:"statementsThisHour"`
	StatementsToday    int64 `json:"statementsToday"`
	StorageBytes       int64 `json:"storageBytes"`
}

type limitsResponse struct {
	SubjectType string        `json:"subjectType"`
	SubjectID   string        `json:"subjectId"`
	Usage       limitUsageDTO `json:"usage"`
	Limits      []limitDTO    `json:"limits"`
}

type putLimitRequest struct {
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
	Metric      string `json:"metric"`
	MaxValue    *int64 `json:"maxValue"`
	Enabled     *bool  `json:"enabled"`
}

func requireOrgManager(ctx context.Context, db *pgxpool.Pool, claims auth.Claims, orgSlug string) error {
	if claims.Role == "admin" || claims.Role == "owner" {
		return nil
	}
	var role string
	err := db.QueryRow(ctx, `
		SELECT om.role
		FROM org_memberships om
		JOIN organisations o ON o.id = om.org_id
		WHERE om.user_id = $1::uuid AND o.slug = $2
	`, claims.UserID, orgSlug).Scan(&role)
	if err == pgx.ErrNoRows || (err == nil && role != "owner" && role != "admin") {
		return fmt.Errorf("access denied")
	}
	if err != nil {
		return fmt.Errorf("check org access failed")
	}
	return nil
}

func subjectInOrg(ctx context.Context, db *pgxpool.Pool, orgSlug, subjectType, subjectID string) (bool, error) {
	var query string
	switch subjectType {
	case limits.SubjectOrganisation:
		query = `SELECT EXISTS(SELECT 1 FROM organisations WHERE id = $1::uuid AND slug = $2)`
	case limits.SubjectProject:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM projects p
				JOIN organisations o ON o.id = p.org_id
				WHERE p.id = $1::uuid AND o.slug = $2
			)`
	case limits.SubjectLRS:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM lrs l
				JOIN projects p ON p.id = l.project_id
				JOIN organisations o ON o.id = p.org_id
				WHERE l.id = $1 AND o.slug = $2
			)`
	case limits.SubjectClient:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM clients c
				JOIN projects p ON p.id = c.project_id
				JOIN organisations o ON o.id = p.org_id
				WHERE c.id = $1::uuid AND o.slug = $2
			)`
	default:
		return false, nil
	}
	var ok bool
	if err := db.QueryRow(ctx, query, subjectID, orgSlug).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func parseLimitSubject(subjectType, subjectID string) error {
	if !limitSubjects[subjectType] {
		return fmt.Errorf("subjectType must be organisation, project, lrs, or client")
	}
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return fmt.Errorf("subjectId is required")
	}
	if subjectType != limits.SubjectLRS {
		if _, err := uuid.Parse(subjectID); err != nil {
			return fmt.Errorf("subjectId must be a uuid")
		}
	}
	return nil
}

func RestGetLimits(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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

		subjectType := strings.TrimSpace(r.URL.Query().Get("subjectType"))
		subjectID := strings.TrimSpace(r.URL.Query().Get("subjectId"))
		if err := parseLimitSubject(subjectType, subjectID); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		belongs, err := subjectInOrg(ctx, db, orgSlug, subjectType, subjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check subject failed"})
			return
		}
		if !belongs {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "subject not found"})
			return
		}

		usage, err := (limits.Checker{DB: db}).Usage(ctx, subjectType, subjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load usage failed"})
			return
		}

		rows, err := db.Query(ctx, `
			SELECT id, subject_type, subject_id, metric, max_value, enabled, updated_at
			FROM usage_limits
			WHERE subject_type = $1 AND subject_id = $2
			ORDER BY metric
		`, subjectType, subjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list limits failed"})
			return
		}
		defer rows.Close()

		items := make([]limitDTO, 0)
		for rows.Next() {
			var item limitDTO
			if err := rows.Scan(&item.ID, &item.SubjectType, &item.SubjectID, &item.Metric, &item.MaxValue, &item.Enabled, &item.UpdatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan limits failed"})
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read limits failed"})
			return
		}

		writeJSON(w, http.StatusOK, limitsResponse{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Usage: limitUsageDTO{
				StatementsTotal:    usage.StatementsTotal,
				StatementsThisHour: usage.StatementsThisHour,
				StatementsToday:    usage.StatementsToday,
				StorageBytes:       usage.StorageBytes,
			},
			Limits: items,
		})
	}
}

func RestPutLimit(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := requireOrgManager(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		var req putLimitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.SubjectType = strings.TrimSpace(req.SubjectType)
		req.SubjectID = strings.TrimSpace(req.SubjectID)
		req.Metric = strings.TrimSpace(req.Metric)
		if err := parseLimitSubject(req.SubjectType, req.SubjectID); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if !limitMetrics[req.Metric] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unknown metric"})
			return
		}
		if req.MaxValue == nil || *req.MaxValue < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "maxValue must be >= 0"})
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		belongs, err := subjectInOrg(ctx, db, orgSlug, req.SubjectType, req.SubjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check subject failed"})
			return
		}
		if !belongs {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "subject not found"})
			return
		}

		var item limitDTO
		err = db.QueryRow(ctx, `
			INSERT INTO usage_limits (subject_type, subject_id, metric, max_value, enabled)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (subject_type, subject_id, metric)
			DO UPDATE SET max_value = EXCLUDED.max_value, enabled = EXCLUDED.enabled, updated_at = now()
			RETURNING id, subject_type, subject_id, metric, max_value, enabled, updated_at
		`, req.SubjectType, req.SubjectID, req.Metric, *req.MaxValue, enabled).Scan(
			&item.ID, &item.SubjectType, &item.SubjectID, &item.Metric, &item.MaxValue, &item.Enabled, &item.UpdatedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "save limit failed"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

func RestDeleteLimit(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := requireOrgManager(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if _, err := uuid.Parse(id); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid limit id"})
			return
		}

		var subjectType, subjectID string
		err := db.QueryRow(ctx, `SELECT subject_type, subject_id FROM usage_limits WHERE id = $1::uuid`, id).Scan(&subjectType, &subjectID)
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "limit not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load limit failed"})
			return
		}
		belongs, err := subjectInOrg(ctx, db, orgSlug, subjectType, subjectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check subject failed"})
			return
		}
		if !belongs {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "limit not found"})
			return
		}
		if _, err := db.Exec(ctx, `DELETE FROM usage_limits WHERE id = $1::uuid`, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete limit failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}
