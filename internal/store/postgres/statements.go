package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type StatementStore struct {
	db *pgxpool.Pool
}

func NewStatementStore(db *pgxpool.Pool) *StatementStore {
	return &StatementStore{db: db}
}

func (s *StatementStore) UpsertStatements(ctx context.Context, stmts []store.Statement) error {
	if len(stmts) == 0 {
		return nil
	}

	// MVP: insert, ignore duplicates by (tenant_id, id)
	// TODO: if duplicate exists with different content -> return ErrAlreadyExists (strict conformance)
	batch := &pgx.Batch{}
	for _, st := range stmts {
		batch.Queue(`
			INSERT INTO statements (
				tenant_id, id, stored_at, actor_mbox, verb_id, object_id, client_id, statement
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (tenant_id, id) DO NOTHING
		`, st.TenantID, st.ID, st.StoredAt, st.ActorMBox, st.VerbID, st.ObjectID, st.ClientID, st.RawJSON)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for range stmts {
		_, err := br.Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *StatementStore) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	var st store.Statement
	var err error
	if tenant == "" {
		err = s.db.QueryRow(ctx, `
			SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, client_id, statement
			FROM statements
			WHERE id=$1
			ORDER BY stored_at DESC
			LIMIT 1
		`, id).Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON)
	} else {
		err = s.db.QueryRow(ctx, `
			SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, client_id, statement
			FROM statements
			WHERE tenant_id=$1 AND id=$2
		`, tenant, id).Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON)
	}

	if err == pgx.ErrNoRows {
		return store.Statement{}, store.ErrNotFound
	}
	if err != nil {
		return store.Statement{}, err
	}
	return st, nil
}

func (s *StatementStore) GetStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	args := make([]any, 0, 8)
	where := "1=1"
	joins := ""
	argn := 1
	if f.OrgID != nil && *f.OrgID != "" {
		joins += " JOIN lrs l ON l.id = statements.tenant_id JOIN projects p ON p.id = l.project_id"
		where += fmt.Sprintf(" AND p.org_id=$%d::uuid", argn)
		args = append(args, *f.OrgID)
		argn++
	}
	if f.TenantID != "" {
		where += fmt.Sprintf(" AND statements.tenant_id=$%d", argn)
		args = append(args, f.TenantID)
		argn++
	}

	// Optional filters
	if f.ActorMBox != nil {
		where += fmt.Sprintf(" AND statements.actor_mbox=$%d", argn)
		args = append(args, *f.ActorMBox)
		argn++
	}
	if f.VerbID != nil {
		where += fmt.Sprintf(" AND statements.verb_id=$%d", argn)
		args = append(args, *f.VerbID)
		argn++
	}
	if f.ObjectID != nil {
		where += fmt.Sprintf(" AND statements.object_id=$%d", argn)
		args = append(args, *f.ObjectID)
		argn++
	}
	if f.StoredSince != nil {
		where += fmt.Sprintf(" AND statements.stored_at >= $%d", argn)
		args = append(args, *f.StoredSince)
		argn++
	}
	if f.StoredUntil != nil {
		where += fmt.Sprintf(" AND statements.stored_at <= $%d", argn)
		args = append(args, *f.StoredUntil)
		argn++
	}

	// Cursor: decode to (stored_at, id)
	if f.Cursor != nil {
		c, err := store.DecodeCursor(*f.Cursor)
		if err != nil {
			return store.StatementPage{}, err
		}
		// stable paging: compare tuple (stored_at, id)
		if f.Ascending {
			where += fmt.Sprintf(" AND (statements.stored_at, statements.id) > ($%d, $%d)", argn, argn+1)
		} else {
			where += fmt.Sprintf(" AND (statements.stored_at, statements.id) < ($%d, $%d)", argn, argn+1)
		}
		args = append(args, c.StoredAt, c.ID)
		argn += 2
	}

	order := "DESC"
	if f.Ascending {
		order = "ASC"
	}

	// fetch one extra to decide NextCursor
	limit := f.Limit + 1
	args = append(args, limit)

	q := fmt.Sprintf(`
		SELECT
			statements.tenant_id,
			statements.id,
			statements.stored_at,
			statements.actor_mbox,
			statements.verb_id,
			statements.object_id,
			statements.client_id,
			statements.statement
		FROM statements
		%s
		WHERE %s
		ORDER BY statements.stored_at %s, statements.id %s
		LIMIT $%d
	`, joins, where, order, order, argn)

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return store.StatementPage{}, err
	}
	defer rows.Close()

	out := make([]store.Statement, 0, f.Limit+1)
	for rows.Next() {
		var st store.Statement
		if err := rows.Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON); err != nil {
			return store.StatementPage{}, err
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return store.StatementPage{}, err
	}

	var next *string
	if len(out) > f.Limit {
		last := out[f.Limit-1]
		token := store.EncodeCursor(store.Cursor{StoredAt: last.StoredAt, ID: last.ID})
		next = &token
		out = out[:f.Limit]
	}

	return store.StatementPage{Statements: out, NextCursor: next}, nil
}
