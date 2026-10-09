package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

func GetActivity(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("activityId")
		if id == "" {
			http.Error(w, "activityId is required", http.StatusBadRequest)
			return
		}
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))
		obj := map[string]any{"objectType": "Activity", "id": id}
		if svc != nil {
			if activities := svc.Bundle(ctx).Activities; activities != nil {
				if def, ok, err := activities.GetActivity(ctx, tenant, id); err == nil && ok && len(def) > 0 {
					var definition any
					if json.Unmarshal(def, &definition) == nil {
						obj["definition"] = definition
					}
				}
			}
		}
		writeJSON(w, http.StatusOK, obj)
	}
}

func GetAgent(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("agent")
		if raw == "" {
			http.Error(w, "agent is required", http.StatusBadRequest)
			return
		}
		var agent map[string]any
		if err := json.Unmarshal([]byte(raw), &agent); err != nil {
			http.Error(w, "agent must be a JSON object", http.StatusBadRequest)
			return
		}
		if err := xapi.ValidateAgentObject(agent); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		person := map[string]any{"objectType": "Person"}
		for _, key := range []string{"name", "mbox", "mbox_sha1sum", "openid"} {
			if v, ok := agent[key]; ok {
				person[key] = []any{v}
			}
		}
		if acc, ok := agent["account"]; ok {
			person["account"] = []any{acc}
		}
		_ = cfg
		_ = svc
		writeJSON(w, http.StatusOK, person)
	}
}
