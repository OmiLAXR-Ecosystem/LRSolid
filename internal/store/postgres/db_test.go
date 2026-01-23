package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func withTestDB(t *testing.T, fn func(ctx context.Context, pool *pgxpool.Pool)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithDatabase("lrsold"),
		tcpostgres.WithUsername("lrsold"),
		tcpostgres.WithPassword("lrsold"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := applyMigrations(t, ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	fn(ctx, pool)
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) error {
	t.Helper()

	// Locate apps/api/migrations relative to this file.
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	migrationsDir := filepath.Clean(filepath.Join(base, "..", "..", "..", "migrations"))

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		_, err = pool.Exec(ctx, string(sqlBytes))
		if err != nil {
			return err
		}
	}
	return nil
}
