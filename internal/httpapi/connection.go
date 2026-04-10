package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

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

func ConnectionStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID("")
		q := r.URL.Query()
		if lrsID := q.Get("lrs"); lrsID != "" {
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

		filter := store.StatementFilter{
			TenantID:    tenant,
			ActorMBox:   optString(q.Get("actor_mbox")),
			VerbID:      optString(q.Get("verb")),
			ObjectID:    optString(q.Get("activity")),
			StoredSince: since,
			StoredUntil: until,
			Limit:       first,
			Ascending:   false,
			Cursor:      afterPtr,
		}

		page, err := svc.QueryStatements(r.Context(), filter)
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
