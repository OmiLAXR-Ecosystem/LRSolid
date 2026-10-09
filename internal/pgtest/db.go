// Package pgtest starts Postgres and applies schema migrations for tests.
package pgtest

import (
	"context"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// WithDB starts a Postgres container, applies *.up.sql migrations, and calls fn.
func WithDB(t *testing.T, fn func(ctx context.Context, pool *pgxpool.Pool)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithDatabase("lrsold"),
		tcpostgres.WithUsername("lrsold"),
		tcpostgres.WithPassword("lrsold"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
		// Bind IPv4 only. Docker Desktop resets connections to the dual-stack publish.
		testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
			hc.PortBindings = nat.PortMap{
				"5432/tcp": {{HostIP: "127.0.0.1", HostPort: "0"}},
			}
		}),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	dsn = forceIPv4(dsn)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := applyMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	fn(ctx, pool)
}

func forceIPv4(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Port() == "" {
		return dsn
	}
	parsed.Host = net.JoinHostPort("127.0.0.1", parsed.Port())
	return parsed.String()
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, file, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(migrationsDir, e.Name()))
	}
	sort.Strings(files)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		// Migration files contain more than one statement. The simple query
		// protocol accepts that; the extended protocol used by pool.Exec does not.
		if _, err := conn.Conn().PgConn().Exec(ctx, string(sqlBytes)).ReadAll(); err != nil {
			return err
		}
	}
	return nil
}
