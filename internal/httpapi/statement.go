package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
)

type statementDetailResponse struct {
	ID        string          `json:"id"`
	StoredAt  string          `json:"storedAt"`
	Timestamp *string         `json:"timestamp,omitempty"`
	ActorMBox *string         `json:"actorMbox,omitempty"`
	VerbID    *string         `json:"verbId,omitempty"`
	ObjectID  *string         `json:"objectId,omitempty"`
	Raw       json.RawMessage `json:"raw"`
}

func StatementByID(_ config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := store.TenantID("")
		if lrsID := r.URL.Query().Get("lrs"); lrsID != "" {
			tenant = store.TenantID(lrsID)
		}
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

		var ts *string
		if st.Timestamp != nil {
			t := st.Timestamp.Format(time.RFC3339Nano)
			ts = &t
		}

		resp := statementDetailResponse{
			ID:        string(st.ID),
			StoredAt:  st.StoredAt.Format(time.RFC3339Nano),
			Timestamp: ts,
			ActorMBox: st.ActorMBox,
			VerbID:    st.VerbID,
			ObjectID:  st.ObjectID,
			Raw:       json.RawMessage(st.RawJSON),
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
