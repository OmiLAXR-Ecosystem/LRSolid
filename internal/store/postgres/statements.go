package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

const voidVerbID = "http://adlnet.gov/expapi/verbs/voided"

type StatementStore struct {
	db *pgxpool.Pool
}

func NewStatementStore(db *pgxpool.Pool) *StatementStore {
	return &StatementStore{db: db}
}

func (s *StatementStore) UpsertStatements(ctx context.Context, stmts []store.Statement) (int, error) {
	if len(stmts) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	inserted := 0
	for _, st := range stmts {
		var insertedID *string
		var existing []byte
		err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO statements (
					tenant_id, id, stored_at, activity_at,
					actor_type, actor_mbox, actor_mbox_sha1sum, actor_openid,
					actor_account_home, actor_account_name, actor_sha256,
					verb_id, object_type, object_id, activity_type,
					registration, context_platform, context_parent_ids, context_group_ids, context_category_ids,
					result_completion, result_success, result_score_scaled, result_duration,
					voided, client_id, statement, activity_ids
				) VALUES (
					$1,$2,$3,$4,
					$5,$6,$7,$8,
					$9,$10,$11,
					$12,$13,$14,$15,
					$16,$17,$18,$19,$20,
					$21,$22,$23,$24,
					$25,$26,$27,$28
				)
				ON CONFLICT (tenant_id, id) DO NOTHING
				RETURNING id
			)
			SELECT (SELECT id FROM ins),
			       (SELECT statement FROM statements WHERE tenant_id = $1 AND id = $2)
		`,
			st.TenantID, st.ID, st.StoredAt, st.Timestamp,
			st.ActorType, st.ActorMBox, st.ActorMboxSHA1Sum, st.ActorOpenID,
			st.ActorAccountHome, st.ActorAccountName, st.ActorSHA256,
			st.VerbID, st.ObjectType, st.ObjectID, st.ActivityType,
			st.Registration, st.ContextPlatform, emptyAsNil(st.ContextParentIDs), emptyAsNil(st.ContextGroupIDs), emptyAsNil(st.ContextCategoryIDs),
			st.ResultCompletion, st.ResultSuccess, st.ResultScoreScaled, st.ResultDuration,
			st.Voided, st.ClientID, st.RawJSON, emptyAsNil(st.ActivityIDs),
		).Scan(&insertedID, &existing)
		if err != nil {
			return 0, err
		}
		if insertedID != nil {
			inserted++
			if err := insertAgents(ctx, tx, st); err != nil {
				return 0, err
			}
			continue
		}
		if !store.StatementsEquivalent(existing, st.RawJSON) {
			return 0, store.ErrAlreadyExists
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return inserted, nil
}

func insertAgents(ctx context.Context, tx pgx.Tx, st store.Statement) error {
	for _, a := range st.Agents {
		if _, err := tx.Exec(ctx, `
			INSERT INTO statement_agents (
				tenant_id, statement_id, mbox, mbox_sha1sum, openid, account_home, account_name
			) VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, st.TenantID, st.ID, a.Mbox, a.MboxSHA1Sum, a.OpenID, a.AccountHome, a.AccountName); err != nil {
			return err
		}
	}
	return nil
}

func emptyAsNil(v []string) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func (s *StatementStore) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	var st store.Statement
	var err error
	if tenant == "" {
		err = s.db.QueryRow(ctx, `
			SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, client_id, statement, voided
			FROM statements
			WHERE id=$1
			ORDER BY stored_at DESC
			LIMIT 1
		`, id).Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON, &st.Voided)
	} else {
		err = s.db.QueryRow(ctx, `
			SELECT tenant_id, id, stored_at, actor_mbox, verb_id, object_id, client_id, statement, voided
			FROM statements
			WHERE tenant_id=$1 AND id=$2
		`, tenant, id).Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON, &st.Voided)
	}
	if err == pgx.ErrNoRows {
		return store.Statement{}, store.ErrNotFound
	}
	if err != nil {
		return store.Statement{}, err
	}
	return st, nil
}

func (s *StatementStore) MarkVoided(ctx context.Context, tenant store.TenantID, id store.StatementID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE statements SET voided = true
		WHERE tenant_id = $1 AND id = $2
	`, tenant, id)
	return err
}

func (s *StatementStore) HasVoidingStatement(ctx context.Context, tenant store.TenantID, id store.StatementID) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM statements
			WHERE tenant_id = $1
			  AND verb_id = $2
			  AND object_id = $3
			  AND object_type = 'StatementRef'
			  AND voided = false
		)
	`, tenant, voidVerbID, string(id)).Scan(&ok)
	return ok, err
}

func (s *StatementStore) GetStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit < 0 {
		f.Limit = 50
	}

	args := make([]any, 0, 12)
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

	switch {
	case f.VoidedStatementID != nil:
		where += fmt.Sprintf(" AND statements.id=$%d AND statements.voided = true", argn)
		args = append(args, *f.VoidedStatementID)
		argn++
	case f.StatementID != nil:
		where += fmt.Sprintf(" AND statements.id=$%d AND statements.voided = false", argn)
		args = append(args, *f.StatementID)
		argn++
	default:
		if !f.IncludeVoided {
			where += " AND statements.voided = false"
		}
	}

	if f.ActorMBox != nil {
		where += fmt.Sprintf(" AND lower(statements.actor_mbox)=lower($%d)", argn)
		args = append(args, *f.ActorMBox)
		argn++
	}
	if f.ActorSHA256 != nil {
		where += fmt.Sprintf(" AND statements.actor_sha256=$%d", argn)
		args = append(args, *f.ActorSHA256)
		argn++
	}
	if f.VerbID != nil {
		where += fmt.Sprintf(" AND statements.verb_id=$%d", argn)
		args = append(args, *f.VerbID)
		argn++
	}
	if f.ObjectID != nil {
		if f.RelatedActivities {
			where += fmt.Sprintf(" AND (statements.object_id=$%d OR $%d = ANY(statements.activity_ids) OR $%d = ANY(statements.context_parent_ids) OR $%d = ANY(statements.context_group_ids) OR $%d = ANY(statements.context_category_ids))", argn, argn, argn, argn, argn)
		} else {
			where += fmt.Sprintf(" AND statements.object_id=$%d AND COALESCE(statements.object_type, 'Activity') IN ('Activity', 'SubStatement')", argn)
		}
		args = append(args, *f.ObjectID)
		argn++
	}
	if f.Registration != nil {
		where += fmt.Sprintf(" AND statements.registration=$%d", argn)
		args = append(args, *f.Registration)
		argn++
	}
	if f.ClientID != nil {
		where += fmt.Sprintf(" AND statements.client_id=$%d::uuid", argn)
		args = append(args, *f.ClientID)
		argn++
	}
	if f.Agent != nil {
		clause, extra, next := agentMatchSQL("statements", "sa", f.Agent, f.RelatedAgents, argn)
		where += " AND " + clause
		args = append(args, extra...)
		argn = next
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

	if f.Cursor != nil {
		c, err := store.DecodeCursor(*f.Cursor)
		if err != nil {
			return store.StatementPage{}, err
		}
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
			statements.statement,
			statements.voided
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
		if err := rows.Scan(&st.TenantID, &st.ID, &st.StoredAt, &st.ActorMBox, &st.VerbID, &st.ObjectID, &st.ClientID, &st.RawJSON, &st.Voided); err != nil {
			return store.StatementPage{}, err
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return store.StatementPage{}, err
	}

	var next *string
	if f.Limit > 0 && len(out) > f.Limit {
		last := out[f.Limit-1]
		token := store.EncodeCursor(store.Cursor{StoredAt: last.StoredAt, ID: last.ID})
		next = &token
		out = out[:f.Limit]
	} else if f.Limit == 0 {
		out = nil
	}

	return store.StatementPage{Statements: out, NextCursor: next}, nil
}

func agentMatchSQL(stmtAlias, agentAlias string, agent *store.AgentRef, related bool, argn int) (string, []any, int) {
	var preds []string
	var args []any
	add := func(col string, v *string, fold bool) {
		if v == nil || *v == "" {
			return
		}
		if fold {
			preds = append(preds, fmt.Sprintf("lower(%s)=lower($%d)", col, argn))
		} else {
			preds = append(preds, fmt.Sprintf("%s=$%d", col, argn))
		}
		args = append(args, *v)
		argn++
	}
	if related {
		add(agentAlias+".mbox", agent.Mbox, true)
		add(agentAlias+".mbox_sha1sum", agent.MboxSHA1Sum, true)
		add(agentAlias+".openid", agent.OpenID, false)
		add(agentAlias+".account_home", agent.AccountHome, false)
		add(agentAlias+".account_name", agent.AccountName, false)
		if len(preds) == 0 {
			return "FALSE", nil, argn
		}
		clause := fmt.Sprintf(`EXISTS (
			SELECT 1 FROM statement_agents %s
			WHERE %s.tenant_id = %s.tenant_id
			  AND %s.statement_id = %s.id
			  AND %s
		)`, agentAlias, agentAlias, stmtAlias, agentAlias, stmtAlias, strings.Join(preds, " AND "))
		return clause, args, argn
	}
	add(stmtAlias+".actor_mbox", agent.Mbox, true)
	add(stmtAlias+".actor_mbox_sha1sum", agent.MboxSHA1Sum, true)
	add(stmtAlias+".actor_openid", agent.OpenID, false)
	add(stmtAlias+".actor_account_home", agent.AccountHome, false)
	add(stmtAlias+".actor_account_name", agent.AccountName, false)
	if len(preds) == 0 {
		return "FALSE", nil, argn
	}
	return strings.Join(preds, " AND "), args, argn
}
