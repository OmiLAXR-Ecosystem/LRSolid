// scripts/create_admin_user.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	var (
		email    = flag.String("email", "", "admin email (required)")
		password = flag.String("password", "", "admin password (required)")
		orgSlug  = flag.String("org", "default", "organisation slug")
		orgName  = flag.String("org-name", "", "organisation name (defaults to slug)")
		role     = flag.String("role", "owner", "org role: owner|admin|member|viewer")
	)
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --email and --password are required")
		os.Exit(2)
	}

	// Default org name from slug if not provided
	if *orgName == "" {
		slug := strings.TrimSpace(*orgSlug)
		if slug == "" {
			slug = "default"
			*orgSlug = slug
		}
		if len(slug) == 1 {
			*orgName = strings.ToUpper(slug)
		} else {
			*orgName = strings.ToUpper(slug[:1]) + slug[1:]
		}
	}

	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "ERROR: PG_DSN is required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	// bcrypt hash
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: bcrypt:", err)
		os.Exit(1)
	}
	hash := string(hashBytes)

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: begin:", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	// 1) Create or get organisation by slug
	var orgID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organisations (slug, name)
		VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, *orgSlug, *orgName).Scan(&orgID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: upsert organisation:", err)
		os.Exit(1)
	}

	// 2) Create or update user by email (set password_hash)
	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		  SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, *email, *email, hash).Scan(&userID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: upsert user:", err)
		os.Exit(1)
	}

	// 3) Ensure org membership with role
	_, err = tx.Exec(ctx, `
		INSERT INTO org_memberships (org_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (org_id, user_id) DO UPDATE
		  SET role = EXCLUDED.role
	`, orgID, userID, *role)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: upsert org_membership:", err)
		os.Exit(1)
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: commit:", err)
		os.Exit(1)
	}

	fmt.Printf("OK: admin user created/updated\n- email: %s\n- user_id: %s\n- org: %s (%s)\n- role: %s\n",
		*email, userID, *orgSlug, orgID, *role)
}
