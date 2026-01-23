package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/<user>/lrsold/apps/api/internal/store"
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
				tenant_id, id, stored_at, actor_mbox, verb_id, object_id, statement
			) VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (tenant_id, id) DO NOTHING
		`, st.TenantID, st.ID, st.StoredAt, st.ActorMBox, st.VerbID, st.ObjectID, st.RawJSON)
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
	err := s.db.QueryRow(ctx, `
		SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, statement
		FROM statements
		WHERE tenant_id=$1 AND id=$2
	`, tenant, id).Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.RawJSON)

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

	args := []any{f.TenantID}
	where := "tenant_id=$1"
	argn := 2

	// Optional filters
	if f.ActorMBox != nil {
		where += fmt.Sprintf(" AND actor_mbox=$%d", argn)
		args = append(args, *f.ActorMBox)
		argn++
	}
	if f.VerbID != nil {
		where += fmt.Sprintf(" AND verb_id=$%d", argn)
		args = append(args, *f.VerbID)
		argn++
	}
	if f.ObjectID != nil {
		where += fmt.Sprintf(" AND object_id=$%d", argn)
		args = append(args, *f.ObjectID)
		argn++
	}
	if f.StoredSince != nil {
		where += fmt.Sprintf(" AND stored_at >= $%d", argn)
		args = append(args, *f.StoredSince)
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
			where += fmt.Sprintf(" AND (stored_at, id) > ($%d, $%d)", argn, argn+1)
		} else {
			where += fmt.Sprintf(" AND (stored_at, id) < ($%d, $%d)", argn, argn+1)
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
		SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, statement
		FROM statements
		WHERE %s
		ORDER BY stored_at %s, id %s
		LIMIT $%d
	`, where, order, order, argn)

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return store.StatementPage{}, err
	}
	defer rows.Close()

	out := make([]store.Statement, 0, f.Limit+1)
	for rows.Next() {
		var st store.Statement
		if err := rows.Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.RawJSON); err != nil {
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
