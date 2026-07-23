package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := flag.String("database", "", "PostgreSQL connection URL")
	migrationsPath := flag.String("path", "migrations", "migration files directory")
	flag.Parse()

	if *databaseURL == "" {
		fatalf("missing -database URL")
	}
	if flag.NArg() != 1 {
		fatalf("usage: go run ./scripts/migrate -database URL [-path DIR] <up|down>")
	}

	driverURL, err := pgxURL(*databaseURL)
	if err != nil {
		fatalf("invalid database URL: %v", err)
	}

	absPath, err := filepath.Abs(*migrationsPath)
	if err != nil {
		fatalf("resolve migrations path: %v", err)
	}
	sourceURL := (&url.URL{Scheme: "file", Path: absPath}).String()

	m, err := migrate.New(sourceURL, driverURL)
	if err != nil {
		fatalf("initialize migrations: %v", err)
	}
	defer m.Close()

	command := flag.Arg(0)
	switch command {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	default:
		fatalf("unknown command %q; expected up or down", command)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("No migration changes to apply.")
		return
	}
	if err != nil {
		fatalf("migrate %s: %v", command, err)
	}

	fmt.Printf("Migration %s completed.\n", command)
}

func pgxURL(databaseURL string) (string, error) {
	switch {
	case strings.HasPrefix(databaseURL, "pgx5://"):
		return databaseURL, nil
	case strings.HasPrefix(databaseURL, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://"), nil
	case strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgresql://"), nil
	default:
		return "", errors.New("URL must use postgres://, postgresql://, or pgx5://")
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "migrate: "+format+"\n", args...)
	os.Exit(1)
}
