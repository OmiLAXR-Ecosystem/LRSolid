package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

func TestXAPIStatementsRouteWithoutAuth_MatchesBothSlashVariants(t *testing.T) {
	cfg := config.Config{
		DefaultTenant: "default",
		MaxPageLimit:  100,
		JWTSecret:     "dev-secret",
	}

	router := NewRouter(cfg, nil, nil, &plugins.Registry{})

	tests := []struct {
		name string
		path string
	}{
		{name: "without trailing slash", path: "/xapi/statements"},
		{name: "with trailing slash", path: "/xapi/statements/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected %d, got %d for path %q", http.StatusUnauthorized, rr.Code, tc.path)
			}
		})
	}
}
