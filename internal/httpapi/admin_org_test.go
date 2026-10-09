package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/limits"
	"github.com/SGoerzen/LRSolid/internal/pgtest"
)

func TestAdminCreateOrg_AppliesDefaultLimits(t *testing.T) {
	pgtest.WithDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		cfg := config.Config{
			DefaultOrgStorageBytes:     5 * 1024 * 1024,
			DefaultOrgStatementsPerDay: 1000,
		}
		org := createOrg(t, pool, cfg, "demo", "Demo")

		var storage, perDay int64
		err := pool.QueryRow(ctx, `
			SELECT max_value FROM usage_limits
			WHERE subject_type = 'organisation' AND subject_id = $1 AND metric = $2
		`, org.ID, limits.MetricStorageBytes).Scan(&storage)
		if err != nil {
			t.Fatalf("storage rule: %v", err)
		}
		if storage != cfg.DefaultOrgStorageBytes {
			t.Fatalf("storage max = %d, want %d", storage, cfg.DefaultOrgStorageBytes)
		}
		err = pool.QueryRow(ctx, `
			SELECT max_value FROM usage_limits
			WHERE subject_type = 'organisation' AND subject_id = $1 AND metric = $2
		`, org.ID, limits.MetricStatementsPerDay).Scan(&perDay)
		if err != nil {
			t.Fatalf("daily rule: %v", err)
		}
		if perDay != 1000 {
			t.Fatalf("statements per day = %d, want 1000", perDay)
		}

		unlimited := createOrg(t, pool, config.Config{}, "open", "Open")
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM usage_limits WHERE subject_id = $1`, unlimited.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("zero config wrote %d rules", n)
		}
	})
}

func createOrg(t *testing.T, pool *pgxpool.Pool, cfg config.Config, slug, name string) restOrg {
	t.Helper()
	body, err := json.Marshal(createOrgRequest{Name: name, Slug: slug})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/org", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		UserID: "00000000-0000-0000-0000-000000000001",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
	rr := httptest.NewRecorder()
	AdminCreateOrg(cfg, pool).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %s: status %d body %s", slug, rr.Code, rr.Body.String())
	}
	var org restOrg
	if err := json.NewDecoder(rr.Body).Decode(&org); err != nil {
		t.Fatal(err)
	}
	if org.ID == "" || org.Slug != slug {
		t.Fatalf("org = %#v", org)
	}
	return org
}
