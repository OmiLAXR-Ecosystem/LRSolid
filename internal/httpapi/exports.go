package httpapi

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	RequireRemoteTargetAboveBytes int64  `json:"requireRemoteTargetAboveBytes"`
	LocalMaxBytes                 int64  `json:"localMaxBytes"`
	DefaultTargetType             string `json:"defaultTargetType"`
	S3UploadBaseURL               string `json:"s3UploadBaseUrl,omitempty"`
	S3PublicBaseURL               string `json:"s3PublicBaseUrl,omitempty"`
	S3AuthToken                   string `json:"s3AuthToken,omitempty"`
	S3PathPrefix                  string `json:"s3PathPrefix"`
}

type updateExportSettingsRequest struct {
	RequireRemoteTargetAboveBytes *int64  `json:"requireRemoteTargetAboveBytes"`
	LocalMaxBytes                 *int64  `json:"localMaxBytes"`
	DefaultTargetType             *string `json:"defaultTargetType"`
	S3UploadBaseURL               *string `json:"s3UploadBaseUrl"`
	S3PublicBaseURL               *string `json:"s3PublicBaseUrl"`
	S3AuthToken                   *string `json:"s3AuthToken"`
	S3PathPrefix                  *string `json:"s3PathPrefix"`
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
	ProjectID *string `json:"projectId,omitempty"`
	LRSID     *string `json:"lrsId,omitempty"`
	ClientID  *string `json:"clientId,omitempty"`
	// StatementWrapper controls whether each exported line is wrapped as:
	// {"metadata": {...}, "statement": {...raw xAPI...}}
	StatementWrapper bool                    `json:"statementWrapper,omitempty"`
	ActorMbox        *fieldConditionReq      `json:"actorMbox,omitempty"`
	VerbID           *fieldConditionReq      `json:"verbId,omitempty"`
	ObjectID         *fieldConditionReq      `json:"objectId,omitempty"`
	StoredAt         *fieldConditionReq      `json:"storedAt,omitempty"`
	Extensions       []extensionConditionReq `json:"extensions,omitempty"`
	Format           string                  `json:"format,omitempty"`
	Compression      string                  `json:"compression,omitempty"`
	TargetType       *string                 `json:"targetType,omitempty"`
	TargetPath       *string                 `json:"targetPath,omitempty"`
}

// statementWhereFilter holds all filter parameters for building a WHERE clause.
type statementWhereFilter struct {
	OrgID            string
	ProjectID        *string
	LRSID            *string
	ClientID         *string
	StatementWrapper bool
	ActorMbox        *fieldConditionReq
	VerbID           *fieldConditionReq
	ObjectID         *fieldConditionReq
	StoredAt         *fieldConditionReq
	Extensions       []extensionConditionReq
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

type exportStatementMetadata struct {
	StatementID      string  `json:"statementId"`
	StoredAt         string  `json:"storedAt"`
	TenantID         string  `json:"tenantId"`
	LRSID            string  `json:"lrsId"`
	LRSName          string  `json:"lrsName"`
	ProjectID        string  `json:"projectId"`
	ProjectSlug      string  `json:"projectSlug"`
	ProjectName      string  `json:"projectName"`
	OrganizationID   string  `json:"organizationId"`
	OrganizationSlug string  `json:"organizationSlug"`
	OrganizationName string  `json:"organizationName"`
	DataStoreID      string  `json:"dataStoreId"`
	DataStoreName    string  `json:"dataStoreName"`
	DataStoreType    string  `json:"dataStoreType"`
	ClientID         *string `json:"clientId,omitempty"`
	ClientName       *string `json:"clientName,omitempty"`
}

type exportWrappedStatement struct {
	Metadata  exportStatementMetadata `json:"metadata"`
	Statement json.RawMessage         `json:"statement"`
}

func ensureExportSettings(ctx context.Context, cfg config.Config, db *pgxpool.Pool, orgID string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO org_export_settings (
			org_id,
			default_target_type,
			s3_upload_base_url,
			s3_public_base_url,
			s3_auth_token,
			s3_path_prefix
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6)
		ON CONFLICT (org_id) DO NOTHING
	`, orgID, cfg.ExportDefaultTargetType, cfg.ExportS3UploadBaseURL, cfg.ExportS3PublicBaseURL, cfg.ExportS3AuthToken, cfg.ExportS3PathPrefix)
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
		if err := ensureExportSettings(ctx, cfg, db, orgID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create export settings failed"})
			return
		}

		var out exportSettingsResponse
		if err := db.QueryRow(ctx, `
			SELECT
				require_remote_target_above_bytes,
				local_max_bytes,
				default_target_type,
				COALESCE(s3_upload_base_url, ''),
				COALESCE(s3_public_base_url, ''),
				COALESCE(s3_auth_token, ''),
				s3_path_prefix
			FROM org_export_settings
			WHERE org_id = $1
		`, orgID).Scan(
			&out.RequireRemoteTargetAboveBytes,
			&out.LocalMaxBytes,
			&out.DefaultTargetType,
			&out.S3UploadBaseURL,
			&out.S3PublicBaseURL,
			&out.S3AuthToken,
			&out.S3PathPrefix,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query export settings failed"})
			return
		}
		out.S3UploadBaseURL = firstNonEmpty(out.S3UploadBaseURL, strings.TrimSpace(cfg.ExportS3UploadBaseURL))
		out.S3PublicBaseURL = firstNonEmpty(out.S3PublicBaseURL, strings.TrimSpace(cfg.ExportS3PublicBaseURL))
		out.S3AuthToken = firstNonEmpty(out.S3AuthToken, strings.TrimSpace(cfg.ExportS3AuthToken))
		out.S3PathPrefix = firstNonEmpty(out.S3PathPrefix, strings.TrimSpace(cfg.ExportS3PathPrefix), "exports")
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
		if err := ensureExportSettings(ctx, cfg, db, orgID); err != nil {
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
		defaultTargetType := normalizeOptionalString(req.DefaultTargetType)
		if defaultTargetType != nil {
			normalized := strings.ToLower(strings.TrimSpace(*defaultTargetType))
			if normalized == "" {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "defaultTargetType must not be empty"})
				return
			}
			if !isAllowedDefaultTargetType(normalized) {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported defaultTargetType"})
				return
			}
			defaultTargetType = &normalized
		}
		s3UploadBaseURL := normalizeTrimmedPointer(req.S3UploadBaseURL)
		s3PublicBaseURL := normalizeTrimmedPointer(req.S3PublicBaseURL)
		s3AuthToken := normalizeTrimmedPointer(req.S3AuthToken)
		s3PathPrefix := normalizeTrimmedPointer(req.S3PathPrefix)

		if _, err := db.Exec(ctx, `
			UPDATE org_export_settings
			SET
				require_remote_target_above_bytes = COALESCE($2, require_remote_target_above_bytes),
				local_max_bytes = COALESCE($3, local_max_bytes),
				default_target_type = COALESCE($4, default_target_type),
				s3_upload_base_url = CASE WHEN $5::text IS NULL THEN s3_upload_base_url ELSE NULLIF($5, '') END,
				s3_public_base_url = CASE WHEN $6::text IS NULL THEN s3_public_base_url ELSE NULLIF($6, '') END,
				s3_auth_token = CASE WHEN $7::text IS NULL THEN s3_auth_token ELSE NULLIF($7, '') END,
				s3_path_prefix = CASE
					WHEN $8::text IS NULL THEN s3_path_prefix
					ELSE COALESCE(NULLIF($8, ''), 'exports')
				END,
				updated_at = now()
			WHERE org_id = $1
		`, orgID, req.RequireRemoteTargetAboveBytes, req.LocalMaxBytes, defaultTargetType, s3UploadBaseURL, s3PublicBaseURL, s3AuthToken, s3PathPrefix); err != nil {
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
		if err := ensureExportSettings(ctx, cfg, db, orgID); err != nil {
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
		if targetType != nil {
			normalized := strings.ToLower(strings.TrimSpace(*targetType))
			if normalized == "local" {
				targetType = nil
			} else {
				targetType = &normalized
			}
		}

		wf := statementWhereFilter{
			OrgID:            orgID,
			ProjectID:        projectID,
			LRSID:            lrsID,
			ClientID:         clientID,
			StatementWrapper: req.StatementWrapper,
			ActorMbox:        req.ActorMbox,
			VerbID:           req.VerbID,
			ObjectID:         req.ObjectID,
			StoredAt:         req.StoredAt,
			Extensions:       req.Extensions,
		}
		estimateRows, estimateBytes, err := estimateStatementExport(ctx, db, wf)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		var requireRemoteAbove int64
		var defaultTargetType string
		var orgS3UploadBaseURL, orgS3PathPrefix string
		if err := db.QueryRow(ctx, `
			SELECT
				require_remote_target_above_bytes,
				default_target_type,
				COALESCE(s3_upload_base_url, ''),
				s3_path_prefix
			FROM org_export_settings
			WHERE org_id = $1
		`, orgID).Scan(&requireRemoteAbove, &defaultTargetType, &orgS3UploadBaseURL, &orgS3PathPrefix); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read export settings failed"})
			return
		}
		effectiveS3UploadBaseURL := firstNonEmpty(strings.TrimSpace(orgS3UploadBaseURL), strings.TrimSpace(cfg.ExportS3UploadBaseURL))
		effectiveS3PathPrefix := firstNonEmpty(strings.TrimSpace(orgS3PathPrefix), strings.TrimSpace(cfg.ExportS3PathPrefix), "exports")
		if targetType == nil {
			dt := strings.ToLower(strings.TrimSpace(defaultTargetType))
			if dt != "" && dt != "local" {
				targetType = &dt
			}
		}
		if targetType != nil && !isAllowedJobTargetType(*targetType) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported targetType"})
			return
		}
		if targetType != nil && targetPath == nil {
			if *targetType == "s3" {
				generated := defaultS3Path(effectiveS3PathPrefix)
				targetPath = &generated
			} else {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "targetPath is required when targetType is set"})
				return
			}
		}
		if targetType != nil && *targetType == "s3" && effectiveS3UploadBaseURL == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "s3 target is selected but no S3 upload URL is configured"})
			return
		}
		if estimateBytes > requireRemoteAbove && targetType == nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "remote target required for this estimated export size"})
			return
		}

		filtersPayload := map[string]any{
			"projectId":        projectID,
			"lrsId":            lrsID,
			"clientId":         clientID,
			"statementWrapper": req.StatementWrapper,
			"actorMbox":        req.ActorMbox,
			"verbId":           req.VerbID,
			"objectId":         req.ObjectID,
			"storedAt":         req.StoredAt,
			"extensions":       req.Extensions,
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
		var targetRaw []byte
		var job exportJobResponse
		err = db.QueryRow(ctx, `
			SELECT id, status, filters::text, format, compression, estimated_rows, estimated_bytes,
			       target_type, target_config::text, artifact_ref, error_message, created_at, started_at, finished_at
			FROM export_jobs
			WHERE org_id = $1 AND id = $2::uuid
		`, orgID, id).Scan(
			&job.ID, &job.Status, &filtersRaw, &job.Format, &job.Compression,
			&job.EstimatedRows, &job.EstimatedBytes,
			&job.TargetType, &targetRaw, &job.ArtifactRef, &job.ErrorMessage,
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
		if job.TargetType != nil && len(targetRaw) > 0 {
			var m map[string]any
			_ = json.Unmarshal(targetRaw, &m)
			if p, ok := m["path"].(string); ok {
				job.TargetPath = &p
			}
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

		setJobStatus := func(status, errMsg string, artifactRef *string) {
			if errMsg != "" {
				_, _ = db.Exec(context.Background(), `
					UPDATE export_jobs SET status = $2, finished_at = now(), error_message = $3 WHERE id = $1::uuid
				`, job.ID, status, errMsg)
				return
			}
			if artifactRef != nil {
				_, _ = db.Exec(context.Background(), `
					UPDATE export_jobs SET status = $2, finished_at = now(), artifact_ref = $3 WHERE id = $1::uuid
				`, job.ID, status, *artifactRef)
				return
			}
			{
				_, _ = db.Exec(context.Background(), `
					UPDATE export_jobs SET status = $2, finished_at = now() WHERE id = $1::uuid
				`, job.ID, status)
			}
		}

		if job.TargetType != nil && *job.TargetType == "s3" {
			var orgS3UploadBaseURL, orgS3PublicBaseURL, orgS3AuthToken string
			if err := db.QueryRow(ctx, `
				SELECT
					COALESCE(s3_upload_base_url, ''),
					COALESCE(s3_public_base_url, ''),
					COALESCE(s3_auth_token, '')
				FROM org_export_settings
				WHERE org_id = $1
			`, orgID).Scan(&orgS3UploadBaseURL, &orgS3PublicBaseURL, &orgS3AuthToken); err != nil {
				setJobStatus("failed", "failed to read s3 settings", nil)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read export settings failed"})
				return
			}
			effectiveS3UploadBaseURL := firstNonEmpty(strings.TrimSpace(orgS3UploadBaseURL), strings.TrimSpace(cfg.ExportS3UploadBaseURL))
			effectiveS3PublicBaseURL := firstNonEmpty(strings.TrimSpace(orgS3PublicBaseURL), strings.TrimSpace(cfg.ExportS3PublicBaseURL))
			effectiveS3AuthToken := firstNonEmpty(strings.TrimSpace(orgS3AuthToken), strings.TrimSpace(cfg.ExportS3AuthToken))

			if effectiveS3UploadBaseURL == "" {
				setJobStatus("failed", "EXPORT_S3_UPLOAD_BASE_URL is not configured", nil)
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "s3 upload base URL is not configured"})
				return
			}
			if job.TargetPath == nil || strings.TrimSpace(*job.TargetPath) == "" {
				setJobStatus("failed", "s3 target path missing on export job", nil)
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "s3 target path missing"})
				return
			}

			artifactURL, err := uploadExportToS3(
				ctx,
				db,
				wf,
				effectiveS3UploadBaseURL,
				effectiveS3PublicBaseURL,
				effectiveS3AuthToken,
				*job.TargetPath,
			)
			if err != nil {
				setJobStatus("failed", err.Error(), nil)
				writeJSON(w, http.StatusBadGateway, errorResponse{Error: "s3 upload failed"})
				return
			}
			setJobStatus("completed", "", &artifactURL)
			http.Redirect(w, r, artifactURL, http.StatusFound)
			return
		}

		filename := "export-" + job.ID + ".ndjson.gz"
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(http.StatusOK)

		gz := gzip.NewWriter(w)

		streamErr := streamStatementsToWriter(ctx, db, where, args, argn, wf.StatementWrapper, gz)
		if err := gz.Close(); err != nil && streamErr == "" {
			streamErr = "compress failed: " + err.Error()
		}

		if streamErr != "" {
			setJobStatus("failed", streamErr, nil)
		} else {
			setJobStatus("completed", "", nil)
		}
	}
}

// filtersToWhereFilter deserialises the JSONB filters blob stored on the job
// back into a statementWhereFilter so the download can replay the same query.
func filtersToWhereFilter(orgID string, raw []byte) statementWhereFilter {
	var m struct {
		ProjectID        *string                 `json:"projectId"`
		LRSID            *string                 `json:"lrsId"`
		ClientID         *string                 `json:"clientId"`
		StatementWrapper bool                    `json:"statementWrapper"`
		ActorMbox        *fieldConditionReq      `json:"actorMbox"`
		VerbID           *fieldConditionReq      `json:"verbId"`
		ObjectID         *fieldConditionReq      `json:"objectId"`
		StoredAt         *fieldConditionReq      `json:"storedAt"`
		Extensions       []extensionConditionReq `json:"extensions"`
	}
	_ = json.Unmarshal(raw, &m)
	return statementWhereFilter{
		OrgID:            orgID,
		ProjectID:        normalizeOptionalString(m.ProjectID),
		LRSID:            normalizeOptionalString(m.LRSID),
		ClientID:         normalizeOptionalString(m.ClientID),
		StatementWrapper: m.StatementWrapper,
		ActorMbox:        m.ActorMbox,
		VerbID:           m.VerbID,
		ObjectID:         m.ObjectID,
		StoredAt:         m.StoredAt,
		Extensions:       m.Extensions,
	}
}

func nullableJSONString(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return string(v)
}

func normalizeTrimmedPointer(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isAllowedDefaultTargetType(v string) bool {
	switch v {
	case "local", "s3", "gdrive", "onedrive", "icloud":
		return true
	default:
		return false
	}
}

func isAllowedJobTargetType(v string) bool {
	switch v {
	case "s3", "gdrive", "onedrive", "icloud":
		return true
	default:
		return false
	}
}

func defaultS3Path(prefix string) string {
	p := strings.Trim(prefix, "/")
	if p == "" {
		p = "exports"
	}
	return fmt.Sprintf("%s/export-%d.ndjson.gz", p, time.Now().UTC().UnixNano())
}

func buildRemoteObjectURL(baseURL, targetPath string) string {
	base := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	path := strings.TrimSpace(strings.TrimLeft(targetPath, "/"))
	if path == "" {
		return base
	}
	return base + "/" + path
}

func streamStatementsToWriter(ctx context.Context, db *pgxpool.Pool, where string, args []any, argn int, wrapStatement bool, w io.Writer) string {
	// Stream in batches using keyset cursor on (stored_at, tenant_id, id)
	var cursorStoredAt time.Time
	var cursorTenantID, cursorID string
	first := true
	const batchSize = 500

	for {
		batchArgs := append(args, cursorStoredAt, cursorTenantID, cursorID, batchSize) //nolint:gocritic
		cursorArgN := argn

		q := `
			SELECT st.stored_at, st.tenant_id, st.id,
			       st.statement::text,
			       l.id::text, l.name,
			       p.id::text, p.slug, p.name,
			       o.id::text, o.slug, o.name,
			       ds.id::text, ds.name, ds.database_type,
			       c.id::text, c.name
			FROM statements st
			JOIN lrs l ON l.id = st.tenant_id
			JOIN projects p ON p.id = l.project_id
			JOIN organisations o ON o.id = p.org_id
			JOIN data_stores ds ON ds.id = l.data_store_id
			LEFT JOIN clients c ON c.id = st.client_id
			WHERE ` + where + `
			  AND (st.stored_at, st.tenant_id::text, st.id) > ($` + strconv.Itoa(cursorArgN) + `, $` + strconv.Itoa(cursorArgN+1) + `, $` + strconv.Itoa(cursorArgN+2) + `)
			ORDER BY st.stored_at ASC, st.tenant_id ASC, st.id ASC
			LIMIT $` + strconv.Itoa(cursorArgN+3)

		if first {
			batchArgs = append(args, batchSize)
			q = `
				SELECT st.stored_at, st.tenant_id, st.id,
				       st.statement::text,
				       l.id::text, l.name,
				       p.id::text, p.slug, p.name,
				       o.id::text, o.slug, o.name,
				       ds.id::text, ds.name, ds.database_type,
				       c.id::text, c.name
				FROM statements st
				JOIN lrs l ON l.id = st.tenant_id
				JOIN projects p ON p.id = l.project_id
				JOIN organisations o ON o.id = p.org_id
				JOIN data_stores ds ON ds.id = l.data_store_id
				LEFT JOIN clients c ON c.id = st.client_id
				WHERE ` + where + `
				ORDER BY st.stored_at ASC, st.tenant_id ASC, st.id ASC
				LIMIT $` + strconv.Itoa(argn)
			first = false
		}

		rows, err := db.Query(ctx, q, batchArgs...)
		if err != nil {
			return "query failed: " + err.Error()
		}

		count := 0
		for rows.Next() {
			var rawStmt string
			var lrsID, lrsName string
			var projectID, projectSlug, projectName string
			var organizationID, organizationSlug, organizationName string
			var dataStoreID, dataStoreName, dataStoreType string
			var clientID, clientName *string
			if err := rows.Scan(
				&cursorStoredAt, &cursorTenantID, &cursorID,
				&rawStmt,
				&lrsID, &lrsName,
				&projectID, &projectSlug, &projectName,
				&organizationID, &organizationSlug, &organizationName,
				&dataStoreID, &dataStoreName, &dataStoreType,
				&clientID, &clientName,
			); err != nil {
				rows.Close()
				return "scan failed: " + err.Error()
			}
			line := rawStmt
			if wrapStatement {
				payload := exportWrappedStatement{
					Metadata: exportStatementMetadata{
						StatementID:      cursorID,
						StoredAt:         cursorStoredAt.UTC().Format(time.RFC3339Nano),
						TenantID:         cursorTenantID,
						LRSID:            lrsID,
						LRSName:          lrsName,
						ProjectID:        projectID,
						ProjectSlug:      projectSlug,
						ProjectName:      projectName,
						OrganizationID:   organizationID,
						OrganizationSlug: organizationSlug,
						OrganizationName: organizationName,
						DataStoreID:      dataStoreID,
						DataStoreName:    dataStoreName,
						DataStoreType:    dataStoreType,
						ClientID:         clientID,
						ClientName:       clientName,
					},
					Statement: json.RawMessage(rawStmt),
				}
				b, err := json.Marshal(payload)
				if err != nil {
					rows.Close()
					return "marshal wrapper failed: " + err.Error()
				}
				line = string(b)
			}
			if _, err := io.WriteString(w, line+"\n"); err != nil {
				rows.Close()
				return "write failed: " + err.Error()
			}
			count++
		}
		rows.Close()
		if rows.Err() != nil {
			return "read failed: " + rows.Err().Error()
		}
		if count < batchSize {
			return ""
		}
	}
}

func uploadExportToS3(
	ctx context.Context,
	db *pgxpool.Pool,
	wf statementWhereFilter,
	uploadBaseURL string,
	publicBaseURL string,
	authToken string,
	targetPath string,
) (string, error) {
	where, args, argn := buildStatementWhere(wf)
	uploadURL := buildRemoteObjectURL(uploadBaseURL, targetPath)
	artifactURL := uploadURL
	if strings.TrimSpace(publicBaseURL) != "" {
		artifactURL = buildRemoteObjectURL(publicBaseURL, targetPath)
	}

	pr, pw := io.Pipe()
	streamDone := make(chan string, 1)
	go func() {
		gz := gzip.NewWriter(pw)
		streamErr := streamStatementsToWriter(ctx, db, where, args, argn, wf.StatementWrapper, gz)
		if err := gz.Close(); err != nil && streamErr == "" {
			streamErr = "compress failed: " + err.Error()
		}
		if streamErr != "" {
			_ = pw.CloseWithError(fmt.Errorf("%s", streamErr))
		} else {
			_ = pw.Close()
		}
		streamDone <- streamErr
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, pr)
	if err != nil {
		_ = pr.Close()
		return "", fmt.Errorf("create s3 upload request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/gzip")
	if token := strings.TrimSpace(authToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = pr.Close()
		streamErr := <-streamDone
		if streamErr != "" {
			return "", fmt.Errorf("%s", streamErr)
		}
		return "", fmt.Errorf("s3 upload request failed: %w", err)
	}
	defer resp.Body.Close()
	streamErr := <-streamDone

	if streamErr != "" {
		return "", fmt.Errorf("%s", streamErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("s3 upload failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return artifactURL, nil
}
