package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

type docKind int

const (
	docState docKind = iota
	docAgentProfile
	docActivityProfile
)

func StateDocument(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return documentHandler(cfg, svc, docState)
}

func AgentProfileDocument(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return documentHandler(cfg, svc, docAgentProfile)
}

func ActivityProfileDocument(cfg config.Config, svc *lrs.Service) http.HandlerFunc {
	return documentHandler(cfg, svc, docActivityProfile)
}

type docRecord struct {
	content     []byte
	contentType string
	etag        string
	updated     time.Time
	exists      bool
}

func documentHandler(cfg config.Config, svc *lrs.Service, kind docKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			http.Error(w, "store unavailable", http.StatusInternalServerError)
			return
		}
		ctx := lrs.WithDataStoreID(r.Context(), dataStoreIDFromRequest(r.Context()))
		tenant := store.TenantID(tenantFromRequest(ctx, cfg))
		q := r.URL.Query()
		agentKey := ""
		if kind == docState || kind == docAgentProfile {
			rawAgent := q.Get("agent")
			if rawAgent == "" {
				http.Error(w, "agent is required", http.StatusBadRequest)
				return
			}
			var agent map[string]any
			if err := json.Unmarshal([]byte(rawAgent), &agent); err != nil {
				http.Error(w, "agent must be a JSON object", http.StatusBadRequest)
				return
			}
			if err := xapi.ValidateAgentObject(agent); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			key, err := xapi.AgentKey(agent)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			agentKey = key
		}
		activityID := q.Get("activityId")
		if (kind == docState || kind == docActivityProfile) && activityID == "" {
			http.Error(w, "activityId is required", http.StatusBadRequest)
			return
		}
		docID := q.Get("stateId")
		if kind != docState {
			docID = q.Get("profileId")
		}
		reg := q.Get("registration")
		regPtr := &reg

		bundle := svc.Bundle(ctx)
		rec, err := loadDoc(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, docID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if docID == "" {
				since, err := parseSince(q.Get("since"))
				if err != nil {
					http.Error(w, "invalid since", http.StatusBadRequest)
					return
				}
				ids, err := listDocIDs(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, since)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				writeJSON(w, http.StatusOK, ids)
				return
			}
			if !rec.exists {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			writeDoc(w, rec)
		case http.MethodPut, http.MethodPost:
			if docID == "" {
				http.Error(w, "document id is required", http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
			if err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			ct := r.Header.Get("Content-Type")
			if ct == "" {
				http.Error(w, "Content-Type is required", http.StatusBadRequest)
				return
			}
			ifNone, hasNone := headerETag(r, "If-None-Match")
			ifMatch, hasMatch := headerETag(r, "If-Match")
			if hasNone && hasMatch {
				http.Error(w, "If-Match and If-None-Match are mutually exclusive", http.StatusBadRequest)
				return
			}
			if r.Method == http.MethodPost && !strings.Contains(strings.ToLower(ct), "application/json") {
				http.Error(w, "POST document must be application/json", http.StatusBadRequest)
				return
			}
			if !rec.exists {
				if hasMatch {
					http.Error(w, "precondition failed", http.StatusPreconditionFailed)
					return
				}
			} else {
				if !hasMatch && !hasNone {
					http.Error(w, "concurrency header required", http.StatusConflict)
					return
				}
				if hasNone && (ifNone == "*" || ifNone == rec.etag) {
					http.Error(w, "precondition failed", http.StatusPreconditionFailed)
					return
				}
				if hasMatch && ifMatch != rec.etag {
					http.Error(w, "precondition failed", http.StatusPreconditionFailed)
					return
				}
				if r.Method == http.MethodPost {
					if !strings.Contains(strings.ToLower(rec.contentType), "application/json") {
						http.Error(w, "cannot merge non-JSON document", http.StatusBadRequest)
						return
					}
					merged, err := mergeJSON(rec.content, body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					body = merged
				}
			}
			var match *store.ETag
			if hasMatch {
				etag := store.ETag(ifMatch)
				match = &etag
			}
			etag, err := putDoc(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, docID, body, ct, match)
			if err != nil {
				if errors.Is(err, store.ErrPreconditionFailed) {
					http.Error(w, "precondition failed", http.StatusPreconditionFailed)
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("ETag", `"`+string(etag)+`"`)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			ifNone, hasNone := headerETag(r, "If-None-Match")
			ifMatch, hasMatch := headerETag(r, "If-Match")
			if hasNone && hasMatch {
				http.Error(w, "If-Match and If-None-Match are mutually exclusive", http.StatusBadRequest)
				return
			}
			if docID == "" {
				if kind != docState {
					http.Error(w, "profileId is required", http.StatusBadRequest)
					return
				}
				ids, err := listDocIDs(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, nil)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				for _, id := range ids {
					_ = deleteDoc(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, id)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if !rec.exists {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if !hasMatch && !hasNone {
				http.Error(w, "concurrency header required", http.StatusConflict)
				return
			}
			if hasNone && (ifNone == "*" || ifNone == rec.etag) {
				http.Error(w, "precondition failed", http.StatusPreconditionFailed)
				return
			}
			if hasMatch && ifMatch != rec.etag {
				http.Error(w, "precondition failed", http.StatusPreconditionFailed)
				return
			}
			if err := deleteDoc(ctx, bundle, kind, tenant, activityID, agentKey, regPtr, docID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func loadDoc(ctx context.Context, bundle store.StoreBundle, kind docKind, tenant store.TenantID, activityID, agentKey string, reg *string, docID string) (docRecord, error) {
	if docID == "" {
		return docRecord{}, nil
	}
	switch kind {
	case docState:
		if bundle.States == nil {
			return docRecord{}, errors.New("state store unavailable")
		}
		doc, ok, err := bundle.States.GetState(ctx, store.StateKey{
			TenantID: tenant, ActivityID: activityID, AgentKey: agentKey, Registration: reg, StateID: docID,
		})
		if err != nil || !ok {
			return docRecord{}, err
		}
		return docRecord{content: doc.Content, contentType: doc.ContentType, etag: string(doc.ETag), updated: doc.UpdatedAt, exists: true}, nil
	case docAgentProfile:
		if bundle.AgentProfiles == nil {
			return docRecord{}, errors.New("profile store unavailable")
		}
		doc, ok, err := bundle.AgentProfiles.GetAgentProfile(ctx, store.AgentProfileKey{TenantID: tenant, AgentKey: agentKey, ProfileID: docID})
		if err != nil || !ok {
			return docRecord{}, err
		}
		return docRecord{content: doc.Content, contentType: doc.ContentType, etag: string(doc.ETag), updated: doc.UpdatedAt, exists: true}, nil
	default:
		if bundle.ActivityProfiles == nil {
			return docRecord{}, errors.New("profile store unavailable")
		}
		doc, ok, err := bundle.ActivityProfiles.GetActivityProfile(ctx, store.ActivityProfileKey{TenantID: tenant, ActivityID: activityID, ProfileID: docID})
		if err != nil || !ok {
			return docRecord{}, err
		}
		return docRecord{content: doc.Content, contentType: doc.ContentType, etag: string(doc.ETag), updated: doc.UpdatedAt, exists: true}, nil
	}
}

func listDocIDs(ctx context.Context, bundle store.StoreBundle, kind docKind, tenant store.TenantID, activityID, agentKey string, reg *string, since *time.Time) ([]string, error) {
	switch kind {
	case docState:
		if bundle.States == nil {
			return nil, errors.New("state store unavailable")
		}
		return bundle.States.ListStateIDs(ctx, tenant, activityID, agentKey, reg, since)
	case docAgentProfile:
		if bundle.AgentProfiles == nil {
			return nil, errors.New("profile store unavailable")
		}
		return bundle.AgentProfiles.ListAgentProfileIDs(ctx, tenant, agentKey, since)
	default:
		if bundle.ActivityProfiles == nil {
			return nil, errors.New("profile store unavailable")
		}
		return bundle.ActivityProfiles.ListActivityProfileIDs(ctx, tenant, activityID, since)
	}
}

func putDoc(ctx context.Context, bundle store.StoreBundle, kind docKind, tenant store.TenantID, activityID, agentKey string, reg *string, docID string, body []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	switch kind {
	case docState:
		return bundle.States.PutState(ctx, store.StateKey{
			TenantID: tenant, ActivityID: activityID, AgentKey: agentKey, Registration: reg, StateID: docID,
		}, body, contentType, ifMatch)
	case docAgentProfile:
		return bundle.AgentProfiles.PutAgentProfile(ctx, store.AgentProfileKey{TenantID: tenant, AgentKey: agentKey, ProfileID: docID}, body, contentType, ifMatch)
	default:
		return bundle.ActivityProfiles.PutActivityProfile(ctx, store.ActivityProfileKey{TenantID: tenant, ActivityID: activityID, ProfileID: docID}, body, contentType, ifMatch)
	}
}

func deleteDoc(ctx context.Context, bundle store.StoreBundle, kind docKind, tenant store.TenantID, activityID, agentKey string, reg *string, docID string) error {
	switch kind {
	case docState:
		return bundle.States.DeleteState(ctx, store.StateKey{
			TenantID: tenant, ActivityID: activityID, AgentKey: agentKey, Registration: reg, StateID: docID,
		})
	case docAgentProfile:
		return bundle.AgentProfiles.DeleteAgentProfile(ctx, store.AgentProfileKey{TenantID: tenant, AgentKey: agentKey, ProfileID: docID})
	default:
		return bundle.ActivityProfiles.DeleteActivityProfile(ctx, store.ActivityProfileKey{TenantID: tenant, ActivityID: activityID, ProfileID: docID})
	}
}

func writeDoc(w http.ResponseWriter, rec docRecord) {
	ct := rec.contentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("ETag", `"`+rec.etag+`"`)
	w.Header().Set("Last-Modified", rec.updated.UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(rec.content)
}

func headerETag(r *http.Request, name string) (string, bool) {
	v := strings.TrimSpace(r.Header.Get(name))
	if v == "" {
		return "", false
	}
	v = strings.TrimPrefix(v, "W/")
	v = strings.Trim(v, `"`)
	return v, true
}

func parseSince(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func mergeJSON(existing, incoming []byte) ([]byte, error) {
	var base map[string]any
	var patch map[string]any
	if err := json.Unmarshal(existing, &base); err != nil {
		return nil, errors.New("existing document is not a JSON object")
	}
	if err := json.Unmarshal(incoming, &patch); err != nil {
		return nil, errors.New("posted document is not a JSON object")
	}
	mergeObjects(base, patch)
	return json.Marshal(base)
}

func mergeObjects(dst, src map[string]any) {
	for k, v := range src {
		if v == nil {
			delete(dst, k)
			continue
		}
		srcObj, srcOK := v.(map[string]any)
		dstObj, dstOK := dst[k].(map[string]any)
		if srcOK && dstOK {
			mergeObjects(dstObj, srcObj)
			continue
		}
		dst[k] = v
	}
}
