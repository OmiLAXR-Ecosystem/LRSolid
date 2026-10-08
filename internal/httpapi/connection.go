package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
)

type connectionEdge struct {
	Cursor string              `json:"cursor"`
	Node   connectionStatement `json:"node"`
}

type connectionPageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor,omitempty"`
}

type connectionResponse struct {
	Edges    []connectionEdge   `json:"edges"`
	PageInfo connectionPageInfo `json:"pageInfo"`
}

type connectionStatement struct {
	ID        string     `json:"id"`
	ActorMBox *string    `json:"actorMbox,omitempty"`
	VerbID    *string    `json:"verbId,omitempty"`
	ObjectID  *string    `json:"objectId,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	StoredAt  time.Time  `json:"storedAt"`
}

func ConnectionStatements(cfg config.Config, svc *lrs.Service, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		orgSlug := q.Get("org")
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

		orgID, err := getOrgID(r.Context(), db, orgSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "query org failed"})
			return
		}
		if orgID == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "organisation not found"})
			return
		}

		tenant := store.TenantID("")
		if lrsID := q.Get("lrs"); lrsID != "" {
			var allowed bool
			if err := db.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1
					FROM lrs l
					JOIN projects p ON p.id = l.project_id
					WHERE l.id = $1::uuid AND p.org_id = $2::uuid
				)
			`, lrsID, orgID).Scan(&allowed); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check lrs failed"})
				return
			}
			if !allowed {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "lrs does not belong to organisation"})
				return
			}
			tenant = store.TenantID(lrsID)
		}

		first := parseLimit(q.Get("first"), 25, cfg.MaxPageLimit)
		after := q.Get("after")
		var afterPtr *string
		if after != "" {
			afterPtr = &after
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

		ctx := r.Context()
		filter := store.StatementFilter{
			TenantID:    tenant,
			OrgID:       &orgID,
			ActorMBox:   optString(q.Get("actor_mbox")),
			VerbID:      optString(q.Get("verb")),
			ObjectID:    optString(q.Get("activity")),
			StoredSince: since,
			StoredUntil: until,
			Limit:       first,
			Ascending:   false,
			Cursor:      afterPtr,
		}
		if tenant == "" && svc != nil && len(svc.ExternalStoreIDs()) > 0 {
			var external int
			if err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM lrs l
				JOIN projects p ON p.id = l.project_id
				WHERE p.org_id = $1::uuid
				  AND l.data_store_id::text = ANY($2::text[])
			`, orgID, svc.ExternalStoreIDs()).Scan(&external); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check data stores failed"})
				return
			}
			if external > 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "this organisation stores statements in an external data store; filter by an LRS"})
				return
			}
		}
		if tenant != "" && svc != nil {
			var dataStoreID string
			if err := db.QueryRow(ctx, `SELECT COALESCE(data_store_id::text, '') FROM lrs WHERE id = $1::uuid`, string(tenant)).Scan(&dataStoreID); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "check data store failed"})
				return
			}
			if svc.HasExternalStore(dataStoreID) {
				ctx = lrs.WithDataStoreID(ctx, dataStoreID)
				filter.OrgID = nil
			}
		}

		page, err := svc.QueryStatements(ctx, filter)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		edges := make([]connectionEdge, 0, len(page.Statements))
		for _, st := range page.Statements {
			cursor := store.EncodeCursor(store.Cursor{StoredAt: st.StoredAt, ID: st.ID})
			edges = append(edges, connectionEdge{
				Cursor: cursor,
				Node: connectionStatement{
					ID:        string(st.ID),
					ActorMBox: st.ActorMBox,
					VerbID:    st.VerbID,
					ObjectID:  st.ObjectID,
					Timestamp: st.Timestamp,
					StoredAt:  st.StoredAt,
				},
			})
		}

		resp := connectionResponse{
			Edges: edges,
			PageInfo: connectionPageInfo{
				HasNextPage: page.NextCursor != nil,
				EndCursor:   page.NextCursor,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func AggregationStatementsNotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "aggregation API not implemented yet"})
	}
}
