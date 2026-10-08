package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
)

type statementDetailMetadataResponse struct {
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

type statementDetailResponse struct {
	ID        string                          `json:"id"`
	StoredAt  string                          `json:"storedAt"`
	Timestamp *string                         `json:"timestamp,omitempty"`
	ActorMBox *string                         `json:"actorMbox,omitempty"`
	VerbID    *string                         `json:"verbId,omitempty"`
	ObjectID  *string                         `json:"objectId,omitempty"`
	Metadata  statementDetailMetadataResponse `json:"metadata"`
	Raw       json.RawMessage                 `json:"raw"`
}

func StatementByID(_ config.Config, svc *lrs.Service, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgSlug := r.URL.Query().Get("org")
		if orgSlug == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing org"})
			return
		}

		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		if err := validateOrgAccess(r.Context(), db, claims, orgSlug); err != nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
			return
		}

		ctx := r.Context()
		tenant := store.TenantID("")
		if lrsID := r.URL.Query().Get("lrs"); lrsID != "" {
			var allowed bool
			var dataStoreID string
			if err := db.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1
					FROM lrs l
					JOIN projects p ON p.id = l.project_id
					JOIN organisations o ON o.id = p.org_id
					WHERE l.id = $1::uuid AND o.slug = $2
				), COALESCE((SELECT data_store_id::text FROM lrs WHERE id = $1::uuid), '')
			`, lrsID, orgSlug).Scan(&allowed, &dataStoreID); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check lrs failed"})
				return
			}
			if !allowed {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "lrs does not belong to organisation"})
				return
			}
			tenant = store.TenantID(lrsID)
			if svc.HasExternalStore(dataStoreID) {
				ctx = lrs.WithDataStoreID(ctx, dataStoreID)
			}
		}
		id := chi.URLParam(r, "id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id"})
			return
		}

		var st store.Statement
		var err error
		if tenant == "" {
			var tenantID string
			err = db.QueryRow(ctx, `
				SELECT st.tenant_id
				FROM statements st
				JOIN lrs l ON l.id = st.tenant_id
				JOIN projects p ON p.id = l.project_id
				JOIN organisations o ON o.id = p.org_id
				WHERE st.id = $1 AND o.slug = $2
				ORDER BY st.stored_at DESC
				LIMIT 1
			`, id, orgSlug).Scan(&tenantID)
			if err == nil {
				tenant = store.TenantID(tenantID)
			} else if err != pgx.ErrNoRows {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
				return
			} else {
				rows, qerr := db.Query(ctx, `
					SELECT l.id::text, COALESCE(l.data_store_id::text, '')
					FROM lrs l
					JOIN projects p ON p.id = l.project_id
					JOIN organisations o ON o.id = p.org_id
					WHERE o.slug = $1
				`, orgSlug)
				if qerr != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
					return
				}
				defer rows.Close()
				for rows.Next() {
					var lrsID, dataStoreID string
					if err := rows.Scan(&lrsID, &dataStoreID); err != nil {
						writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
						return
					}
					if !svc.HasExternalStore(dataStoreID) {
						continue
					}
					found, ferr := svc.GetStatementByID(lrs.WithDataStoreID(ctx, dataStoreID), store.TenantID(lrsID), store.StatementID(id))
					if ferr == nil {
						st = found
						tenant = store.TenantID(lrsID)
						ctx = lrs.WithDataStoreID(ctx, dataStoreID)
						break
					}
					if !errors.Is(ferr, store.ErrNotFound) {
						writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
						return
					}
				}
				if err := rows.Err(); err != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
					return
				}
				if tenant == "" {
					writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
					return
				}
			}
		}
		if st.ID == "" {
			st, err = svc.GetStatementByID(ctx, tenant, store.StatementID(id))
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "lookup failed"})
				return
			}
		}

		var ts *string
		if st.Timestamp != nil {
			t := st.Timestamp.Format(time.RFC3339Nano)
			ts = &t
		}

		var md statementDetailMetadataResponse
		if err := db.QueryRow(r.Context(), `
			SELECT
				l.id::text, l.name,
				p.id::text, p.slug, p.name,
				o.id::text, o.slug, o.name,
				ds.id::text, ds.name, ds.database_type
			FROM lrs l
			JOIN projects p ON p.id = l.project_id
			JOIN organisations o ON o.id = p.org_id
			JOIN data_stores ds ON ds.id = l.data_store_id
			WHERE l.id = $1::uuid AND o.slug = $2
		`, st.TenantID, orgSlug).Scan(
			&md.LRSID, &md.LRSName,
			&md.ProjectID, &md.ProjectSlug, &md.ProjectName,
			&md.OrganizationID, &md.OrganizationSlug, &md.OrganizationName,
			&md.DataStoreID, &md.DataStoreName, &md.DataStoreType,
		); err != nil {
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "metadata lookup failed"})
			return
		}
		md.TenantID = string(st.TenantID)
		if st.ClientID != nil && *st.ClientID != "" {
			var clientID, clientName string
			if err := db.QueryRow(r.Context(), `SELECT id::text, name FROM clients WHERE id = $1::uuid`, *st.ClientID).Scan(&clientID, &clientName); err == nil {
				md.ClientID = &clientID
				md.ClientName = &clientName
			}
		}

		resp := statementDetailResponse{
			ID:        string(st.ID),
			StoredAt:  st.StoredAt.Format(time.RFC3339Nano),
			Timestamp: ts,
			ActorMBox: st.ActorMBox,
			VerbID:    st.VerbID,
			ObjectID:  st.ObjectID,
			Metadata:  md,
			Raw:       json.RawMessage(st.RawJSON),
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
