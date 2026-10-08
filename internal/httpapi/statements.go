package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/limits"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
)

type postStatementsRequest struct {
	Statements []json.RawMessage `json:"statements"`
	// optional: allow single statement
	Statement json.RawMessage `json:"statement"`
}

func PostStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))
		clientID := clientIDFromRequest(ctx)

		rawList, err := parseStatementsPayload(r)
		if err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(rawList) == 0 {
			http.Error(w, "missing statement(s)", http.StatusBadRequest)
			return
		}

		ids, err := svc.IngestStatements(ctx, tenant, clientID, rawList)
		if err != nil {
			var exceeded *limits.ExceededError
			if errors.As(err, &exceeded) {
				if sec := exceeded.RetryAfter(time.Now()); sec > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(sec))
				}
				status := http.StatusRequestEntityTooLarge
				if exceeded.Metric == limits.MetricStatementsPerHour || exceeded.Metric == limits.MetricStatementsPerDay {
					status = http.StatusTooManyRequests
				}
				http.Error(w, exceeded.Error(), status)
				return
			}
			status := http.StatusBadRequest
			if errors.Is(err, store.ErrAlreadyExists) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ids)
	}
}

func parseStatementsPayload(r *http.Request) ([]json.RawMessage, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil
	}

	// xAPI-compatible: raw array body.
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err == nil {
		if len(arr) == 0 {
			return nil, nil
		}
		return arr, nil
	}

	// JSON object body: either wrapper ({statement|statements}) or a raw statement object.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err == nil {
		if raw, ok := obj["statements"]; ok {
			var wrapped []json.RawMessage
			if err := json.Unmarshal(raw, &wrapped); err != nil {
				return nil, fmt.Errorf("invalid statements wrapper: %w", err)
			}
			return wrapped, nil
		}
		if raw, ok := obj["statement"]; ok {
			return []json.RawMessage{raw}, nil
		}
		// Treat any other JSON object as a single raw statement (xAPI spec shape).
		return []json.RawMessage{json.RawMessage(body)}, nil
	}

	// Compatibility fallback (TinCan/legacy): statement(s)=<json>, even if
	// Content-Type is missing or incorrect.
	if bytes.Contains(body, []byte("statement=")) || bytes.Contains(body, []byte("statements=")) {
		form, err := url.ParseQuery(string(body))
		if err == nil {
			if s := strings.TrimSpace(form.Get("statement")); s != "" {
				return []json.RawMessage{json.RawMessage(s)}, nil
			}
			if s := strings.TrimSpace(form.Get("statements")); s != "" {
				var wrapped []json.RawMessage
				if err := json.Unmarshal([]byte(s), &wrapped); err != nil {
					return nil, fmt.Errorf("invalid form statements: %w", err)
				}
				return wrapped, nil
			}
		}
	}

	// Fallback for proper form-encoded payloads.
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		form, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("invalid form data: %w", err)
		}
		if s := strings.TrimSpace(form.Get("statement")); s != "" {
			return []json.RawMessage{json.RawMessage(s)}, nil
		}
		if s := strings.TrimSpace(form.Get("statements")); s != "" {
			var wrapped []json.RawMessage
			if err := json.Unmarshal([]byte(s), &wrapped); err != nil {
				return nil, fmt.Errorf("invalid form statements: %w", err)
			}
			return wrapped, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("invalid json payload")
}

func GetStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))

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

		page, err := svc.QueryStatements(ctx, filter)
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
