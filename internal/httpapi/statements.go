package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/limits"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

func PostStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return writeStatements(cfg, svc, false)
}

func PutStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return writeStatements(cfg, svc, true)
}

func writeStatements(cfg config.Config, svc *lrs.Service, put bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			http.Error(w, "statements unavailable", http.StatusInternalServerError)
			return
		}
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))
		clientID := clientIDFromRequest(ctx)
		q := r.URL.Query()
		if put {
			if strings.TrimSpace(q.Get("statementId")) == "" {
				http.Error(w, "statementId is required", http.StatusBadRequest)
				return
			}
		} else if q.Get("statementId") != "" {
			http.Error(w, "statementId is not valid on POST", http.StatusBadRequest)
			return
		}

		rawList, parts, err := readStatementRequest(r, cfg.XAPILegacyCompat, put)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(rawList) == 0 {
			http.Error(w, "missing statement(s)", http.StatusBadRequest)
			return
		}
		if put && len(rawList) != 1 {
			http.Error(w, "PUT accepts a single statement", http.StatusBadRequest)
			return
		}
		for _, raw := range rawList {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := xapi.CheckAttachments(m, parts); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := verifySignatures(m, parts, versionFromRequest(ctx)); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		home := requestBaseURL(r)
		opts := lrs.IngestOptions{
			Version:          versionFromRequest(ctx),
			AuthorityName:    keyIDFromRequest(ctx),
			AuthorityHome:    home,
			DefineActivities: defineFromRequest(ctx),
		}
		if put {
			opts.StatementID = q.Get("statementId")
		}
		ids, err := svc.Ingest(ctx, tenant, clientID, rawList, opts)
		if err != nil {
			writeIngestError(w, err)
			return
		}
		for sha, part := range parts {
			if err := svc.SaveAttachment(ctx, tenant, sha, part.ContentType, part.Body); err != nil {
				http.Error(w, "attachment storage failed", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set(xapi.HeaderConsistentThrough, xapiStamp(time.Now()))
		if put {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ids)
	}
}

func GetStatements(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			http.Error(w, "statements unavailable", http.StatusInternalServerError)
			return
		}
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))
		filter, single, limitZero, err := parseStatementQuery(r, cfg, tenant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mineOnlyFromRequest(ctx) {
			filter.ClientID = clientIDFromRequest(ctx)
		}
		q := r.URL.Query()
		format := q.Get("format")
		if format == "" {
			format = "exact"
		}
		attachments := strings.EqualFold(q.Get("attachments"), "true")

		if limitZero {
			filter.Limit = 1
		}
		page, err := svc.QueryStatements(ctx, filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lookup := func(id string) (json.RawMessage, bool) {
			if storeActivities := svc.Bundle(ctx).Activities; storeActivities != nil {
				def, ok, err := storeActivities.GetActivity(ctx, tenant, id)
				if err != nil || !ok {
					return nil, false
				}
				return def, true
			}
			return nil, false
		}
		out := make([]json.RawMessage, 0, len(page.Statements))
		for _, st := range page.Statements {
			formatted, ferr := xapi.ApplyFormat(st.RawJSON, format, r.Header.Get("Accept-Language"), lookup)
			if ferr != nil {
				http.Error(w, ferr.Error(), http.StatusBadRequest)
				return
			}
			out = append(out, formatted)
		}

		w.Header().Set(xapi.HeaderConsistentThrough, xapiStamp(time.Now()))
		if single {
			if len(out) == 0 {
				http.Error(w, "statement not found", http.StatusNotFound)
				return
			}
			writeStatementBody(w, r, svc, tenant, out[:1], true, attachments)
			return
		}
		if limitZero {
			out = nil
			if len(page.Statements) > 0 {
				page.NextCursor = nil
				more := moreURL(r, "", true)
				writeStatementResult(w, r, svc, tenant, nil, more, attachments, cfg.XAPILegacyCompat)
				return
			}
		}
		var more string
		if page.NextCursor != nil {
			more = moreURL(r, *page.NextCursor, false)
		}
		writeStatementResult(w, r, svc, tenant, out, more, attachments, cfg.XAPILegacyCompat)
	}
}

func writeStatementResult(w http.ResponseWriter, r *http.Request, svc *lrs.Service, tenant store.TenantID, stmts []json.RawMessage, more string, attachments, legacy bool) {
	body := map[string]any{"statements": stmts}
	if stmts == nil {
		body["statements"] = []json.RawMessage{}
	}
	if more != "" {
		body["more"] = more
	}
	if legacy && more != "" {
		if u, err := url.Parse(more); err == nil {
			if c := u.Query().Get("cursor"); c != "" {
				body["cursor"] = c
			}
		}
	}
	raw, _ := json.Marshal(body)
	if attachments {
		writeStatementBody(w, r, svc, tenant, []json.RawMessage{raw}, false, true)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func writeStatementBody(w http.ResponseWriter, r *http.Request, svc *lrs.Service, tenant store.TenantID, docs []json.RawMessage, single, attachments bool) {
	if !attachments {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(docs[0])
		return
	}
	var mw *multipart.Writer
	body := &bytes.Buffer{}
	mw = multipart.NewWriter(body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", "application/json")
	part, _ := mw.CreatePart(header)
	_, _ = part.Write(docs[0])
	for _, doc := range docs {
		for _, att := range listedAttachments(doc) {
			payload, err := svc.ReadAttachment(tenant, att.sha)
			if err != nil {
				continue
			}
			h := textproto.MIMEHeader{}
			h.Set("Content-Type", att.contentType)
			h.Set("Content-Transfer-Encoding", "binary")
			h.Set(xapi.HeaderHash, att.sha)
			p, err := mw.CreatePart(h)
			if err != nil {
				continue
			}
			_, _ = p.Write(payload)
		}
	}
	_ = mw.Close()
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", mw.Boundary()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
	_ = r
	_ = single
}

type listedAttachment struct {
	sha         string
	contentType string
}

func listedAttachments(raw json.RawMessage) []listedAttachment {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if stmts, ok := m["statements"].([]any); ok {
		var out []listedAttachment
		for _, item := range stmts {
			b, _ := json.Marshal(item)
			out = append(out, listedAttachments(b)...)
		}
		return out
	}
	arr, _ := m["attachments"].([]any)
	out := make([]listedAttachment, 0, len(arr))
	for _, item := range arr {
		am, _ := item.(map[string]any)
		sha, _ := am["sha2"].(string)
		ct, _ := am["contentType"].(string)
		if sha == "" {
			continue
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		out = append(out, listedAttachment{sha: strings.ToLower(sha), contentType: ct})
	}
	return out
}

func readStatementRequest(r *http.Request, legacy, put bool) ([]json.RawMessage, map[string]xapi.AttachmentPart, error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "multipart/mixed") {
		return readMultipartStatements(r)
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		return nil, nil, err
	}
	body = bytes.TrimSpace(body)
	if strings.Contains(ct, "application/x-www-form-urlencoded") || (legacy && (bytes.Contains(body, []byte("statement=")) || bytes.Contains(body, []byte("statements=")))) {
		if !legacy && !strings.Contains(ct, "application/x-www-form-urlencoded") {
			return nil, nil, fmt.Errorf("unsupported content type")
		}
		if strings.Contains(ct, "application/x-www-form-urlencoded") && !legacy {
			return nil, nil, fmt.Errorf("form submissions require XAPI_LEGACY_COMPAT")
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid form data")
		}
		if s := strings.TrimSpace(form.Get("statement")); s != "" {
			return []json.RawMessage{json.RawMessage(s)}, nil, nil
		}
		if s := strings.TrimSpace(form.Get("statements")); s != "" {
			var wrapped []json.RawMessage
			if err := json.Unmarshal([]byte(s), &wrapped); err != nil {
				return nil, nil, fmt.Errorf("invalid form statements")
			}
			return wrapped, nil, nil
		}
		return nil, nil, fmt.Errorf("missing statement(s)")
	}
	if ct != "" && !strings.Contains(ct, "application/json") && !legacy {
		return nil, nil, fmt.Errorf("content type must be application/json or multipart/mixed")
	}
	if len(body) == 0 {
		return nil, nil, nil
	}
	if body[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, nil, fmt.Errorf("invalid json")
		}
		if put {
			return nil, nil, fmt.Errorf("PUT accepts a single statement")
		}
		return arr, nil, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, nil, fmt.Errorf("invalid json")
	}
	if raw, ok := obj["statements"]; ok && legacy {
		var wrapped []json.RawMessage
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, nil, fmt.Errorf("invalid statements wrapper")
		}
		return wrapped, nil, nil
	}
	if raw, ok := obj["statement"]; ok && legacy {
		return []json.RawMessage{raw}, nil, nil
	}
	if _, ok := obj["statements"]; ok && !legacy {
		return nil, nil, fmt.Errorf("POST body must be a statement or an array of statements")
	}
	return []json.RawMessage{json.RawMessage(body)}, nil, nil
}

func readMultipartStatements(r *http.Request) ([]json.RawMessage, map[string]xapi.AttachmentPart, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, nil, fmt.Errorf("invalid multipart content type")
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	var stmts []json.RawMessage
	parts := map[string]xapi.AttachmentPart{}
	first := true
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("invalid multipart body")
		}
		payload, err := io.ReadAll(p)
		if err != nil {
			return nil, nil, err
		}
		if first {
			first = false
			raw := bytes.TrimSpace(payload)
			if len(raw) == 0 {
				return nil, nil, fmt.Errorf("missing statement")
			}
			if raw[0] == '[' {
				if err := json.Unmarshal(raw, &stmts); err != nil {
					return nil, nil, fmt.Errorf("invalid statement json")
				}
			} else {
				stmts = []json.RawMessage{raw}
			}
			continue
		}
		sum := sha256.Sum256(payload)
		sha := hex.EncodeToString(sum[:])
		headerSHA := strings.ToLower(strings.TrimSpace(p.Header.Get(xapi.HeaderHash)))
		if headerSHA != "" && headerSHA != sha {
			return nil, nil, fmt.Errorf("attachment hash header does not match content")
		}
		ct := p.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		parts[sha] = xapi.AttachmentPart{SHA2: sha, ContentType: ct, Body: payload}
	}
	return stmts, parts, nil
}

func verifySignatures(statement map[string]any, parts map[string]xapi.AttachmentPart, version string) error {
	atts, _ := statement["attachments"].([]any)
	for _, item := range atts {
		m, _ := item.(map[string]any)
		usage, _ := m["usageType"].(string)
		if usage != xapi.AttachmentSignature {
			continue
		}
		sha, _ := m["sha2"].(string)
		part, ok := parts[strings.ToLower(sha)]
		if !ok {
			return &xapi.ValidationError{Reason: "signature attachment content is required"}
		}
		if err := xapi.VerifySignature(statement, strings.TrimSpace(string(part.Body)), xapi.Major(version) == "2", time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func parseStatementQuery(r *http.Request, cfg config.Config, tenant store.TenantID) (store.StatementFilter, bool, bool, error) {
	q := r.URL.Query()
	allowed := map[string]bool{
		"statementId": true, "voidedStatementId": true, "agent": true, "verb": true,
		"activity": true, "registration": true, "related_activities": true, "related_agents": true,
		"since": true, "until": true, "limit": true, "format": true, "attachments": true,
		"ascending": true, "cursor": true,
	}
	for key := range q {
		if !allowed[key] {
			return store.StatementFilter{}, false, false, fmt.Errorf("unknown parameter %s", key)
		}
	}
	statementID := strings.TrimSpace(q.Get("statementId"))
	voidedID := strings.TrimSpace(q.Get("voidedStatementID"))
	if q.Has("voidedStatementId") {
		voidedID = strings.TrimSpace(q.Get("voidedStatementId"))
	}
	single := statementID != "" || voidedID != ""
	if statementID != "" && voidedID != "" {
		return store.StatementFilter{}, false, false, fmt.Errorf("statementId and voidedStatementId are mutually exclusive")
	}
	if single {
		for key := range q {
			switch key {
			case "statementId", "voidedStatementId", "format", "attachments":
			default:
				return store.StatementFilter{}, false, false, fmt.Errorf("%s cannot be used with statementId", key)
			}
		}
	}
	format := q.Get("format")
	if format == "" {
		format = "exact"
	}
	switch format {
	case "ids", "exact", "canonical":
	default:
		return store.StatementFilter{}, false, false, fmt.Errorf("format must be ids, exact, or canonical")
	}
	if raw := q.Get("attachments"); raw != "" && raw != "true" && raw != "false" {
		return store.StatementFilter{}, false, false, fmt.Errorf("attachments must be true or false")
	}
	filter := store.StatementFilter{TenantID: tenant, Ascending: false}
	if statementID != "" {
		filter.StatementID = &statementID
		filter.Limit = 1
		return filter, true, false, nil
	}
	if voidedID != "" {
		filter.VoidedStatementID = &voidedID
		filter.Limit = 1
		return filter, true, false, nil
	}
	if raw := q.Get("ascending"); raw != "" {
		switch raw {
		case "true":
			filter.Ascending = true
		case "false":
			filter.Ascending = false
		default:
			return store.StatementFilter{}, false, false, fmt.Errorf("ascending must be true or false")
		}
	}
	limitZero := false
	limit := cfg.MaxPageLimit
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return store.StatementFilter{}, false, false, fmt.Errorf("invalid limit")
		}
		if n == 0 {
			limitZero = true
			limit = 0
		} else if n > cfg.MaxPageLimit {
			limit = cfg.MaxPageLimit
		} else {
			limit = n
		}
	}
	if cfg.MaxPageLimit <= 0 {
		limit = 100
	}
	filter.Limit = limit
	if v := strings.TrimSpace(q.Get("verb")); v != "" {
		filter.VerbID = &v
	}
	if v := strings.TrimSpace(q.Get("activity")); v != "" {
		filter.ObjectID = &v
	}
	if v := strings.TrimSpace(q.Get("registration")); v != "" {
		filter.Registration = &v
	}
	relAct, err := optBool(q.Get("related_activities"))
	if err != nil {
		return store.StatementFilter{}, false, false, fmt.Errorf("related_activities must be true or false")
	}
	relAgent, err := optBool(q.Get("related_agents"))
	if err != nil {
		return store.StatementFilter{}, false, false, fmt.Errorf("related_agents must be true or false")
	}
	if relAct && filter.ObjectID == nil {
		return store.StatementFilter{}, false, false, fmt.Errorf("related_activities requires activity")
	}
	filter.RelatedActivities = relAct
	if agent := q.Get("agent"); agent != "" {
		ref, err := xapi.ParseAgentFilter(agent)
		if err != nil {
			return store.StatementFilter{}, false, false, err
		}
		filter.Agent = &ref
	} else if relAgent {
		return store.StatementFilter{}, false, false, fmt.Errorf("related_agents requires agent")
	}
	filter.RelatedAgents = relAgent
	if raw := q.Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return store.StatementFilter{}, false, false, fmt.Errorf("invalid since")
		}
		filter.StoredSince = &t
	}
	if raw := q.Get("until"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return store.StatementFilter{}, false, false, fmt.Errorf("invalid until")
		}
		filter.StoredUntil = &t
	}
	if c := q.Get("cursor"); c != "" {
		filter.Cursor = &c
	}
	return filter, false, limitZero, nil
}

func optBool(v string) (bool, error) {
	switch v {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, fmt.Errorf("invalid boolean")
	}
}

func moreURL(r *http.Request, cursor string, dropLimit bool) string {
	q := r.URL.Query()
	if dropLimit {
		q.Del("limit")
		q.Del("cursor")
	} else {
		q.Set("cursor", cursor)
	}
	return "/xapi/statements?" + q.Encode()
}

func writeIngestError(w http.ResponseWriter, err error) {
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
	var ve *xapi.ValidationError
	if errors.As(err, &ve) {
		http.Error(w, ve.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, lrs.ErrCannotVoidVoiding) {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if errors.Is(err, store.ErrAlreadyExists) {
		http.Error(w, "statement conflict", http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return scheme + "://" + r.Host
}

func xapiStamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func parseStatementsPayload(r *http.Request) ([]json.RawMessage, error) {
	stmts, _, err := readStatementRequest(r, true, false)
	return stmts, err
}

func optString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
