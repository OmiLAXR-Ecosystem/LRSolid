package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "", "migration direction: up or down")
	path := flag.String("path", "migrations", "directory containing migrations")
	flag.Parse()

	if *direction != "up" && *direction != "down" {
		fmt.Fprintln(os.Stderr, "ERROR: --direction must be up or down")
		os.Exit(2)
	}

	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "ERROR: PG_DSN is required")
		os.Exit(2)
	}

	absPath, err := filepath.Abs(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: resolve migrations path:", err)
		os.Exit(1)
	}

	m, err := migrate.New("file://"+filepath.ToSlash(absPath), dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: initialize migrations:", err)
		os.Exit(1)
	}
	defer m.Close()

	if *direction == "up" {
		err = m.Up()
	} else {
		err = m.Steps(-1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("No migrations to apply.")
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: migrate", *direction+":", err)
		os.Exit(1)
	}

	fmt.Println("Migration", *direction, "completed.")
}
