package httpapi

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
)

type exportSettingsResponse struct {
	RequireRemoteTargetAboveBytes int64 `json:"requireRemoteTargetAboveBytes"`
	LocalMaxBytes                 int64 `json:"localMaxBytes"`
}

type updateExportSettingsRequest struct {
	RequireRemoteTargetAboveBytes *int64 `json:"requireRemoteTargetAboveBytes"`
	LocalMaxBytes                 *int64 `json:"localMaxBytes"`
}

type fieldConditionReq struct {
	Op      string `json:"operator"`
	Value   string `json:"value"`
	ValueTo string `json:"valueTo,omitempty"`
}

type extensionConditionReq struct {
	Key   string `json:"key"`
	Op    string `json:"operator"`
	Value string `json:"value"`
}

type createExportJobRequest struct {
	ProjectID  *string                 `json:"projectId,omitempty"`
	LRSID      *string                 `json:"lrsId,omitempty"`
	ClientID   *string                 `json:"clientId,omitempty"`
	ActorMbox  *fieldConditionReq      `json:"actorMbox,omitempty"`
	VerbID     *fieldConditionReq      `json:"verbId,omitempty"`
	ObjectID   *fieldConditionReq      `json:"objectId,omitempty"`
	StoredAt   *fieldConditionReq      `json:"storedAt,omitempty"`
	Extensions []extensionConditionReq `json:"extensions,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Compression string                 `json:"compression,omitempty"`
	TargetType  *string                `json:"targetType,omitempty"`
	TargetPath  *string                `json:"targetPath,omitempty"`
}

// statementWhereFilter holds all filter parameters for building a WHERE clause.
type statementWhereFilter struct {
	OrgID      string
	ProjectID  *string
	LRSID      *string
	ClientID   *string
	ActorMbox  *fieldConditionReq
	VerbID     *fieldConditionReq
	ObjectID   *fieldConditionReq
	StoredAt   *fieldConditionReq
	Extensions []extensionConditionReq
}

// buildStatementWhere constructs the WHERE clause and args for statement queries.
// Returns (whereClause, args, nextArgIndex).
func buildStatementWhere(f statementWhereFilter) (string, []any, int) {
	args := []any{f.OrgID}
	where := "p.org_id = $1"
	n := 2

	add := func(clause string, vals ...any) {
		where += " AND " + clause
		args = append(args, vals...)
		n += len(vals)
	}

	if f.ProjectID != nil {
		add(fmt.Sprintf("p.id = $%d", n), *f.ProjectID)
	}
	if f.LRSID != nil {
		add(fmt.Sprintf("st.tenant_id = $%d", n), *f.LRSID)
	}
	if f.ClientID != nil {
		add(fmt.Sprintf("st.client_id = $%d::uuid", n), *f.ClientID)
	}
	if c := f.ActorMbox; c != nil && c.Value != "" {
		where += " AND " + textConditionSQL("st.actor_mbox", c, n)
		if c.Op == "contains" {
			args = append(args, "%"+c.Value+"%")
		} else {
			args = append(args, c.Value)
		}
		n++
	}
	if c := f.VerbID; c != nil && c.Value != "" {
		where += " AND " + textConditionSQL("st.verb_id", c, n)
		if c.Op == "contains" {
			args = append(args, "%"+c.Value+"%")
		} else {
			args = append(args, c.Value)
		}
		n++
	}
	if c := f.ObjectID; c != nil && c.Value != "" {
		where += " AND " + textConditionSQL("st.object_id", c, n)
		if c.Op == "contains" {
			args = append(args, "%"+c.Value+"%")
		} else {
			args = append(args, c.Value)
		}
		n++
	}
	if c := f.StoredAt; c != nil && c.Value != "" {
		if c.Op == "between" && c.ValueTo != "" {
			add(fmt.Sprintf("st.stored_at BETWEEN $%d::timestamptz AND $%d::timestamptz", n, n+1), c.Value, c.ValueTo)
		} else {
			op := timestampOp(c.Op)
			add(fmt.Sprintf("st.stored_at %s $%d::timestamptz", op, n), c.Value)
		}
	}
	for _, ext := range f.Extensions {
		if ext.Key == "" || ext.Value == "" {
			continue
		}
		// Check in context.extensions, result.extensions, object.definition.extensions
		clause, val := extensionConditionSQL(ext, n)
		where += " AND " + clause
		args = append(args, ext.Key, val)
		n += 2
	}

	return where, args, n
}

func textConditionSQL(col string, c *fieldConditionReq, n int) string {
	switch c.Op {
	case "neq":
		return fmt.Sprintf("%s <> $%d", col, n)
	case "contains":
		return fmt.Sprintf("%s ILIKE $%d", col, n)
	default: // eq
		return fmt.Sprintf("%s = $%d", col, n)
	}
}

func timestampOp(op string) string {
	switch op {
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "gt":
		return ">"
	case "gte":
		return ">="
	default: // eq
		return "="
	}
}

func extensionConditionSQL(ext extensionConditionReq, n int) (clause, val string) {
	val = ext.Value
	paths := []string{
		fmt.Sprintf("(st.statement -> 'context' -> 'extensions' ->> $%d)", n),
		fmt.Sprintf("(st.statement -> 'result' -> 'extensions' ->> $%d)", n),
		fmt.Sprintf("(st.statement -> 'object' -> 'definition' -> 'extensions' ->> $%d)", n),
	}
	var conds []string
	valN := n + 1
	for _, p := range paths {
		switch ext.Op {
		case "neq":
			conds = append(conds, fmt.Sprintf("%s <> $%d", p, valN))
		case "contains":
			conds = append(conds, fmt.Sprintf("%s ILIKE $%d", p, valN))
		case "lt":
			conds = append(conds, fmt.Sprintf("(%s)::numeric < ($%d)::numeric", p, valN))
		case "lte":
			conds = append(conds, fmt.Sprintf("(%s)::numeric <= ($%d)::numeric", p, valN))
		case "gt":
			conds = append(conds, fmt.Sprintf("(%s)::numeric > ($%d)::numeric", p, valN))
		case "gte":
			conds = append(conds, fmt.Sprintf("(%s)::numeric >= ($%d)::numeric", p, valN))
		default: // eq
			conds = append(conds, fmt.Sprintf("%s = $%d", p, valN))
		}
	}
	if ext.Op == "contains" {
		val = "%" + ext.Value + "%"
	}
	clause = "(" + strings.Join(conds, " OR ") + ")"
	return clause, val
}

type exportJobResponse struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	Filters        any        `json:"filters"`
	Format         string     `json:"format"`
	Compression    string     `json:"compression"`
	EstimatedRows  int64      `json:"estimatedRows"`
	EstimatedBytes int64      `json:"estimatedBytes"`
	TargetType     *string    `json:"targetType,omitempty"`
	TargetPath     *string    `json:"targetPath,omitempty"`
	ArtifactRef    *string    `json:"artifactRef,omitempty"`
	ErrorMessage   *string    `json:"errorMessage,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

type exportJobListResponse struct {
	Items []exportJobResponse `json:"items"`
}

func ensureExportSettings(ctx context.Context, db *pgxpool.Pool, orgID string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO org_export_settings (org_id)
		VALUES ($1)
		ON CONFLICT (org_id) DO NOTHING
	`, orgID)
	return err
}

func RestGetExportSettings(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		if err := ensureExportSettings(ctx, db, orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create export settings failed"})
			return
		}

		var out exportSettingsResponse
		if err := db.QueryRow(ctx, `
			SELECT require_remote_target_above_bytes, local_max_bytes
			FROM org_export_settings
			WHERE org_id = $1
		`, orgID).Scan(&out.RequireRemoteTargetAboveBytes, &out.LocalMaxBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query export settings failed"})
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func RestUpdateExportSettings(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := auth.ClaimsFromContext(ctx)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		if claims.Role != "admin" && claims.Role != "owner" {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin or owner required"})
			return
		}
		orgSlug := orgFromRequest(r, cfg)
		if err := validateOrgAccess(ctx, db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}
		orgID, err := getOrgID(ctx, db, orgSlug)
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		if err := ensureExportSettings(ctx, db, orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create export settings failed"})
			return
		}

		var req updateExportSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		if req.RequireRemoteTargetAboveBytes != nil && *req.RequireRemoteTargetAboveBytes < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "requireRemoteTargetAboveBytes must be >= 0"})
			return
		}
		if req.LocalMaxBytes != nil && *req.LocalMaxBytes < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "localMaxBytes must be >= 0"})
			return
		}

		if _, err := db.Exec(ctx, `
			UPDATE org_export_settings
			SET
				require_remote_target_above_bytes = COALESCE($2, require_remote_target_above_bytes),
				local_max_bytes = COALESCE($3, local_max_bytes),
				updated_at = now()
			WHERE org_id = $1
		`, orgID, req.RequireRemoteTargetAboveBytes, req.LocalMaxBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "update export settings failed"})
			return
		}

		RestGetExportSettings(cfg, db).ServeHTTP(w, r)
	}
}

func RestCreateStatementExport(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		if err := ensureExportSettings(ctx, db, orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create export settings failed"})
			return
		}

		var req createExportJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}

		format := strings.TrimSpace(req.Format)
		if format == "" {
			format = "ndjson"
		}
		if format != "ndjson" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported format"})
			return
		}
		compression := strings.TrimSpace(req.Compression)
		if compression == "" {
			compression = "gzip"
		}
		if compression != "gzip" && compression != "none" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported compression"})
			return
		}

		projectID := normalizeOptionalString(req.ProjectID)
		lrsID := normalizeOptionalString(req.LRSID)
		clientID := normalizeOptionalString(req.ClientID)
		targetType := normalizeOptionalString(req.TargetType)
		targetPath := normalizeOptionalString(req.TargetPath)
		if targetType != nil && targetPath == nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "targetPath is required when targetType is set"})
			return
		}

		wf := statementWhereFilter{
			OrgID:      orgID,
			ProjectID:  projectID,
			LRSID:      lrsID,
			ClientID:   clientID,
			ActorMbox:  req.ActorMbox,
			VerbID:     req.VerbID,
			ObjectID:   req.ObjectID,
			StoredAt:   req.StoredAt,
			Extensions: req.Extensions,
		}
		estimateRows, estimateBytes, err := estimateStatementExport(ctx, db, wf)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		var requireRemoteAbove int64
		if err := db.QueryRow(ctx, `
			SELECT require_remote_target_above_bytes
			FROM org_export_settings
			WHERE org_id = $1
		`, orgID).Scan(&requireRemoteAbove); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read export settings failed"})
			return
		}
		if estimateBytes > requireRemoteAbove && targetType == nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "remote target required for this estimated export size"})
			return
		}

		filtersPayload := map[string]any{
			"projectId":  projectID,
			"lrsId":      lrsID,
			"clientId":   clientID,
			"actorMbox":  req.ActorMbox,
			"verbId":     req.VerbID,
			"objectId":   req.ObjectID,
			"storedAt":   req.StoredAt,
			"extensions": req.Extensions,
		}
		targetConfig := map[string]any{
			"path": targetPath,
		}
		filtersJSON, _ := json.Marshal(filtersPayload)
		var targetConfigJSON []byte
		if targetType != nil {
			targetConfigJSON, _ = json.Marshal(targetConfig)
		}

		var out exportJobResponse
		var filtersRaw []byte
		var targetRaw []byte
		err = db.QueryRow(ctx, `
			INSERT INTO export_jobs (
				org_id, created_by_user, status, filters, format, compression,
				estimated_rows, estimated_bytes, target_type, target_config
			)
			VALUES ($1, $2::uuid, 'queued', $3::jsonb, $4, $5, $6, $7, $8, $9::jsonb)
			RETURNING id, status, filters::text, format, compression, estimated_rows, estimated_bytes,
				target_type, target_config::text, artifact_ref, error_message, created_at, started_at, finished_at
		`, orgID, claims.UserID, string(filtersJSON), format, compression, estimateRows, estimateBytes, targetType, nullableJSONString(targetConfigJSON)).Scan(
			&out.ID, &out.Status, &filtersRaw, &out.Format, &out.Compression, &out.EstimatedRows, &out.EstimatedBytes,
			&out.TargetType, &targetRaw, &out.ArtifactRef, &out.ErrorMessage, &out.CreatedAt, &out.StartedAt, &out.FinishedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create export job failed"})
			return
		}
		_ = json.Unmarshal(filtersRaw, &out.Filters)
		if out.TargetType != nil && len(targetRaw) > 0 {
			var m map[string]any
			_ = json.Unmarshal(targetRaw, &m)
			if p, ok := m["path"].(string); ok {
				out.TargetPath = &p
			}
		}

		writeJSON(w, http.StatusCreated, out)
	}
}

func RestListExportJobs(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"), 25, 200)

		rows, err := db.Query(ctx, `
			SELECT id, status, filters::text, format, compression, estimated_rows, estimated_bytes,
				   target_type, target_config::text, artifact_ref, error_message, created_at, started_at, finished_at
			FROM export_jobs
			WHERE org_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`, orgID, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list export jobs failed"})
			return
		}
		defer rows.Close()

		items := make([]exportJobResponse, 0, limit)
		for rows.Next() {
			var out exportJobResponse
			var filtersRaw []byte
			var targetRaw []byte
			if err := rows.Scan(
				&out.ID, &out.Status, &filtersRaw, &out.Format, &out.Compression, &out.EstimatedRows, &out.EstimatedBytes,
				&out.TargetType, &targetRaw, &out.ArtifactRef, &out.ErrorMessage, &out.CreatedAt, &out.StartedAt, &out.FinishedAt,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan export jobs failed"})
				return
			}
			_ = json.Unmarshal(filtersRaw, &out.Filters)
			if out.TargetType != nil && len(targetRaw) > 0 {
				var m map[string]any
				_ = json.Unmarshal(targetRaw, &m)
				if p, ok := m["path"].(string); ok {
					out.TargetPath = &p
				}
			}
			items = append(items, out)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read export jobs failed"})
			return
		}
		writeJSON(w, http.StatusOK, exportJobListResponse{Items: items})
	}
}

func RestGetExportJob(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "job id required"})
			return
		}
		var out exportJobResponse
		var filtersRaw []byte
		var targetRaw []byte
		err = db.QueryRow(ctx, `
			SELECT id, status, filters::text, format, compression, estimated_rows, estimated_bytes,
				   target_type, target_config::text, artifact_ref, error_message, created_at, started_at, finished_at
			FROM export_jobs
			WHERE org_id = $1 AND id = $2::uuid
		`, orgID, id).Scan(
			&out.ID, &out.Status, &filtersRaw, &out.Format, &out.Compression, &out.EstimatedRows, &out.EstimatedBytes,
			&out.TargetType, &targetRaw, &out.ArtifactRef, &out.ErrorMessage, &out.CreatedAt, &out.StartedAt, &out.FinishedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "export job not found"})
			return
		}
		_ = json.Unmarshal(filtersRaw, &out.Filters)
		if out.TargetType != nil && len(targetRaw) > 0 {
			var m map[string]any
			_ = json.Unmarshal(targetRaw, &m)
			if p, ok := m["path"].(string); ok {
				out.TargetPath = &p
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func estimateStatementExport(ctx context.Context, db *pgxpool.Pool, f statementWhereFilter) (rows int64, bytes int64, err error) {
	where, args, _ := buildStatementWhere(f)
	q := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(AVG(pg_column_size(st.statement))::bigint, 1024)
		FROM statements st
		JOIN lrs l ON l.id = st.tenant_id
		JOIN projects p ON p.id = l.project_id
		WHERE ` + where
	var avgSize int64
	if err := db.QueryRow(ctx, q, args...).Scan(&rows, &avgSize); err != nil {
		return 0, 0, fmt.Errorf("estimate query failed")
	}
	if avgSize <= 0 {
		avgSize = 1024
	}
	return rows, rows * avgSize, nil
}

func RestDownloadExportJob(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
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
		if err != nil || orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "job id required"})
			return
		}

		// Load job and verify it belongs to this org
		var filtersRaw []byte
		var job exportJobResponse
		err = db.QueryRow(ctx, `
			SELECT id, status, filters::text, format, compression, estimated_rows, estimated_bytes,
			       target_type, artifact_ref, error_message, created_at, started_at, finished_at
			FROM export_jobs
			WHERE org_id = $1 AND id = $2::uuid
		`, orgID, id).Scan(
			&job.ID, &job.Status, &filtersRaw, &job.Format, &job.Compression,
			&job.EstimatedRows, &job.EstimatedBytes,
			&job.TargetType, &job.ArtifactRef, &job.ErrorMessage,
			&job.CreatedAt, &job.StartedAt, &job.FinishedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "export job not found"})
			return
		}

		// If completed with remote artifact: redirect
		if job.Status == "completed" && job.ArtifactRef != nil && *job.ArtifactRef != "" {
			http.Redirect(w, r, *job.ArtifactRef, http.StatusFound)
			return
		}

		// Parse stored filters back into filter struct
		wf := filtersToWhereFilter(orgID, filtersRaw)

		where, args, argn := buildStatementWhere(wf)

		// Mark job as running
		if _, err := db.Exec(ctx, `
			UPDATE export_jobs SET status = 'running', started_at = now() WHERE id = $1::uuid
		`, job.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update job status"})
			return
		}

		setJobStatus := func(status, errMsg string) {
			if errMsg != "" {
				_, _ = db.Exec(context.Background(), `
					UPDATE export_jobs SET status = $2, finished_at = now(), error_message = $3 WHERE id = $1::uuid
				`, job.ID, status, errMsg)
			} else {
				_, _ = db.Exec(context.Background(), `
					UPDATE export_jobs SET status = $2, finished_at = now() WHERE id = $1::uuid
				`, job.ID, status)
			}
		}

		filename := "export-" + job.ID + ".ndjson.gz"
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(http.StatusOK)

		gz := gzip.NewWriter(w)

		// Stream in batches using keyset cursor on (stored_at, tenant_id, id)
		var cursorStoredAt time.Time
		var cursorTenantID, cursorID string
		first := true
		const batchSize = 500
		streamErr := ""

		for {
			batchArgs := append(args, cursorStoredAt, cursorTenantID, cursorID, batchSize) //nolint:gocritic
			cursorArgN := argn

			q := `
				SELECT st.stored_at, st.tenant_id, st.id, st.statement::text
				FROM statements st
				JOIN lrs l ON l.id = st.tenant_id
				JOIN projects p ON p.id = l.project_id
				WHERE ` + where + `
				  AND (st.stored_at, st.tenant_id::text, st.id) > ($` + strconv.Itoa(cursorArgN) + `, $` + strconv.Itoa(cursorArgN+1) + `, $` + strconv.Itoa(cursorArgN+2) + `)
				ORDER BY st.stored_at ASC, st.tenant_id ASC, st.id ASC
				LIMIT $` + strconv.Itoa(cursorArgN+3)

			if first {
				batchArgs = append(args, batchSize)
				q = `
					SELECT st.stored_at, st.tenant_id, st.id, st.statement::text
					FROM statements st
					JOIN lrs l ON l.id = st.tenant_id
					JOIN projects p ON p.id = l.project_id
					WHERE ` + where + `
					ORDER BY st.stored_at ASC, st.tenant_id ASC, st.id ASC
					LIMIT $` + strconv.Itoa(argn)
				first = false
			}

			rows, err := db.Query(ctx, q, batchArgs...)
			if err != nil {
				streamErr = "query failed: " + err.Error()
				break
			}

			count := 0
			for rows.Next() {
				var rawStmt string
				if err := rows.Scan(&cursorStoredAt, &cursorTenantID, &cursorID, &rawStmt); err != nil {
					rows.Close()
					streamErr = "scan failed: " + err.Error()
					break
				}
				_, _ = gz.Write([]byte(rawStmt))
				_, _ = gz.Write([]byte("\n"))
				count++
			}
			rows.Close()
			if rows.Err() != nil {
				streamErr = "read failed: " + rows.Err().Error()
				break
			}
			if streamErr != "" || count < batchSize {
				break
			}
		}

		_ = gz.Close()

		if streamErr != "" {
			setJobStatus("failed", streamErr)
		} else {
			setJobStatus("completed", "")
		}
	}
}

// filtersToWhereFilter deserialises the JSONB filters blob stored on the job
// back into a statementWhereFilter so the download can replay the same query.
func filtersToWhereFilter(orgID string, raw []byte) statementWhereFilter {
	var m struct {
		ProjectID  *string                 `json:"projectId"`
		LRSID      *string                 `json:"lrsId"`
		ClientID   *string                 `json:"clientId"`
		ActorMbox  *fieldConditionReq      `json:"actorMbox"`
		VerbID     *fieldConditionReq      `json:"verbId"`
		ObjectID   *fieldConditionReq      `json:"objectId"`
		StoredAt   *fieldConditionReq      `json:"storedAt"`
		Extensions []extensionConditionReq `json:"extensions"`
	}
	_ = json.Unmarshal(raw, &m)
	return statementWhereFilter{
		OrgID:      orgID,
		ProjectID:  normalizeOptionalString(m.ProjectID),
		LRSID:      normalizeOptionalString(m.LRSID),
		ClientID:   normalizeOptionalString(m.ClientID),
		ActorMbox:  m.ActorMbox,
		VerbID:     m.VerbID,
		ObjectID:   m.ObjectID,
		StoredAt:   m.StoredAt,
		Extensions: m.Extensions,
	}
}

func nullableJSONString(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return string(v)
}
