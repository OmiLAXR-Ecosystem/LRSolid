package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/plugins"
	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

func TestAboutDoesNotRequireAuth(t *testing.T) {
	router := NewRouter(config.Config{JWTSecret: "dev-secret", MaxPageLimit: 100}, nil, nil, &plugins.Registry{}, nil, store.NewRegistry())
	req := httptest.NewRequest(http.MethodGet, "/xapi/about", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get(xapi.HeaderVersion) != xapi.Version200 {
		t.Fatalf("version header %s", rr.Header().Get(xapi.HeaderVersion))
	}
	var body struct {
		Version []string `json:"version"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Version) != 2 || body.Version[0] != xapi.Version103 || body.Version[1] != xapi.Version200 {
		t.Fatalf("versions %+v", body.Version)
	}
}

func TestVersionHeaderRequired(t *testing.T) {
	h := XAPIVersion(config.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/xapi/statements", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/xapi/statements", nil)
	req.Header.Set(xapi.HeaderVersion, "2.0.0")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Header().Get(xapi.HeaderVersion) != xapi.Version200 {
		t.Fatalf("header %s", rr.Header().Get(xapi.HeaderVersion))
	}
}

func TestScopes(t *testing.T) {
	scopes := []byte(`{"statements:write":true,"state":true}`)
	ok, mine := xapiPermitted(scopes, http.MethodPut, "/xapi/statements")
	if !ok || mine {
		t.Fatalf("write permitted=%v mine=%v", ok, mine)
	}
	ok, mine = xapiPermitted(scopes, http.MethodGet, "/xapi/statements")
	if !ok || !mine {
		t.Fatalf("write implies mine read, permitted=%v mine=%v", ok, mine)
	}
	ok, _ = xapiPermitted(scopes, http.MethodPut, "/xapi/activities/state")
	if !ok {
		t.Fatal("expected state write")
	}
	ok, _ = xapiPermitted(scopes, http.MethodGet, "/xapi/agents/profile")
	if ok {
		t.Fatal("state scope must not allow profiles")
	}
	all := []byte(`{"all/read":true}`)
	ok, mine = xapiPermitted(all, http.MethodGet, "/xapi/statements")
	if !ok || mine {
		t.Fatalf("all/read permitted=%v mine=%v", ok, mine)
	}
	ok, _ = xapiPermitted(all, http.MethodPost, "/xapi/statements")
	if ok {
		t.Fatal("all/read must not allow writes")
	}
}
