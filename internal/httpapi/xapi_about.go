package httpapi

import (
	"net/http"

	"github.com/SGoerzen/LRSolid/internal/xapi"
)

func GetAbout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		negotiated := xapi.Version200
		if raw := r.Header.Get(xapi.HeaderVersion); raw != "" {
			got, ok := xapi.Negotiate(raw)
			if !ok {
				http.Error(w, "X-Experience-API-Version is unsupported", http.StatusBadRequest)
				return
			}
			negotiated = got
		}
		w.Header().Set(xapi.HeaderVersion, negotiated)
		writeJSON(w, http.StatusOK, map[string]any{
			"version": []string{xapi.Version103, xapi.Version200},
		})
	}
}
