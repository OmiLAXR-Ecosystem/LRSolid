package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/SGoerzen/LRSolid/apps/api/internal/config"
	"github.com/SGoerzen/LRSolid/apps/api/internal/lrs"
	"github.com/SGoerzen/LRSolid/apps/api/internal/store"
)

type postStatementsRequest struct {
	Statements []json.RawMessage `json:"statements"`
	// optional: allow single statement
	Statement json.RawMessage `json:"statement"`
}

func PostStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID(cfg.DefaultTenant)

		var req postStatementsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		var rawList []json.RawMessage
		if len(req.Statements) > 0 {
			rawList = req.Statements
		} else if len(req.Statement) > 0 {
			rawList = []json.RawMessage{req.Statement}
		} else {
			http.Error(w, "missing statement(s)", http.StatusBadRequest)
			return
		}

		if err := svc.IngestStatements(r.Context(), tenant, rawList); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
	}
}

func GetStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID(cfg.DefaultTenant)

		q := r.URL.Query()
		limit := cfg.MaxPageLimit
		if s := q.Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 {
				if n > cfg.MaxPageLimit {
					n = cfg.MaxPageLimit
				}
				limit = n
			}
		}

		var since *time.Time
		if s := q.Get("since"); s != "" {
			t, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				http.Error(w, "invalid since", http.StatusBadRequest)
				return
			}
			since = &t
		}

		var cursor *string
		if c := q.Get("cursor"); c != "" {
			cursor = &c
		}

		filter := store.StatementFilter{
			TenantID:    tenant,
			VerbID:      optString(q.Get("verb")),
			ObjectID:    optString(q.Get("activity")),
			ActorMBox:   optString(q.Get("actor_mbox")),
			StoredSince: since,
			Limit:       limit,
			Ascending:   false, // default newest first
			Cursor:      cursor,
		}

		page, err := svc.QueryStatements(r.Context(), filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return raw JSON statements (typical for xAPI)
		out := make([]json.RawMessage, 0, len(page.Statements))
		for _, s := range page.Statements {
			out = append(out, json.RawMessage(s.RawJSON))
		}

		resp := map[string]any{
			"statements": out,
		}
		if page.NextCursor != nil {
			resp["cursor"] = *page.NextCursor
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func optString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
