package limits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

const (
	SubjectOrganisation = "organisation"
	SubjectProject      = "project"
	SubjectLRS          = "lrs"
	SubjectClient       = "client"

	MetricStatementsTotal   = "statements_total"
	MetricStatementsPerHour = "statements_per_hour"
	MetricStatementsPerDay  = "statements_per_day"
	MetricStorageBytes      = "storage_bytes"
)

// Checker enforces usage_limits against the control-plane counters.
// Counters for the primary database lag until the rollup worker runs.
type Checker struct {
	DB *pgxpool.Pool
}

type Subject struct {
	Type string
	ID   string
}

type Usage struct {
	StatementsTotal    int64
	StatementsThisHour int64
	StatementsToday    int64
	StorageBytes       int64
}

type Rule struct {
	SubjectType string
	SubjectID   string
	Metric      string
	Max         int64
}

// ExceededError is returned when an incoming batch would cross a rule.
type ExceededError struct {
	SubjectType string
	SubjectID   string
	Metric      string
	Max         int64
	Current     int64
}

func (e *ExceededError) Error() string {
	return fmt.Sprintf("usage limit exceeded: %s %s (%d/%d)", e.SubjectType, e.Metric, e.Current, e.Max)
}

// RetryAfter is the seconds until the window resets. Totals and storage have none.
func (e *ExceededError) RetryAfter(now time.Time) int {
	now = now.UTC()
	var next time.Time
	switch e.Metric {
	case MetricStatementsPerHour:
		next = now.Truncate(time.Hour).Add(time.Hour)
	case MetricStatementsPerDay:
		next = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	default:
		return 0
	}
	sec := int(next.Sub(now).Seconds()) + 1
	if sec < 1 {
		return 1
	}
	return sec
}

func (u Usage) current(metric string) (int64, bool) {
	switch metric {
	case MetricStatementsTotal:
		return u.StatementsTotal, true
	case MetricStatementsPerHour:
		return u.StatementsThisHour, true
	case MetricStatementsPerDay:
		return u.StatementsToday, true
	case MetricStorageBytes:
		return u.StorageBytes, true
	default:
		return 0, false
	}
}

// Check rejects the batch when any enabled rule on the chain
// client → LRS → project → organisation would be exceeded.
// An unknown tenant has no rules and is allowed.
func (c Checker) Check(ctx context.Context, tenant store.TenantID, clientID *string, count int, bytes int64) error {
	if c.DB == nil || count <= 0 || tenant == "" {
		return nil
	}
	subjects, err := c.chain(ctx, string(tenant), clientID)
	if err != nil || len(subjects) == 0 {
		return err
	}
	rules, err := c.enabledRules(ctx, subjects)
	if err != nil || len(rules) == 0 {
		return err
	}
	usage, err := c.loadUsage(ctx, subjects)
	if err != nil {
		return err
	}
	if ex := evaluate(rules, usage, count, bytes); ex != nil {
		return ex
	}
	return nil
}

// Usage reports the counters for one subject. A missing row is zero usage.
func (c Checker) Usage(ctx context.Context, subjectType, subjectID string) (Usage, error) {
	usage, err := c.loadUsage(ctx, []Subject{{Type: subjectType, ID: subjectID}})
	if err != nil {
		return Usage{}, err
	}
	return usage[Subject{Type: subjectType, ID: subjectID}], nil
}

func evaluate(rules []Rule, usage map[Subject]Usage, count int, bytes int64) *ExceededError {
	for _, rule := range rules {
		cur, ok := usage[Subject{Type: rule.SubjectType, ID: rule.SubjectID}].current(rule.Metric)
		if !ok {
			continue
		}
		add := int64(count)
		if rule.Metric == MetricStorageBytes {
			add = bytes
		}
		if cur+add > rule.Max {
			return &ExceededError{
				SubjectType: rule.SubjectType,
				SubjectID:   rule.SubjectID,
				Metric:      rule.Metric,
				Max:         rule.Max,
				Current:     cur,
			}
		}
	}
	return nil
}

func (c Checker) chain(ctx context.Context, tenant string, clientID *string) ([]Subject, error) {
	var projectID, orgID string
	err := c.DB.QueryRow(ctx, `
		SELECT l.project_id::text, p.org_id::text
		FROM lrs l
		JOIN projects p ON p.id = l.project_id
		WHERE l.id = $1
	`, tenant).Scan(&projectID, &orgID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	subjects := make([]Subject, 0, 4)
	if orgID != "" {
		subjects = append(subjects, Subject{Type: SubjectOrganisation, ID: orgID})
	}
	if projectID != "" {
		subjects = append(subjects, Subject{Type: SubjectProject, ID: projectID})
	}
	subjects = append(subjects, Subject{Type: SubjectLRS, ID: tenant})

	if clientID != nil && *clientID != "" {
		if _, err := uuid.Parse(*clientID); err != nil {
			return subjects, nil
		}
		var id string
		err = c.DB.QueryRow(ctx, `
			SELECT id::text
			FROM clients
			WHERE id = $1::uuid
			  AND (
			    lrs_id = $2
			    OR (lrs_id IS NULL AND project_id = $3::uuid)
			  )
		`, *clientID, tenant, projectID).Scan(&id)
		if err == nil && id != "" {
			subjects = append(subjects, Subject{Type: SubjectClient, ID: id})
		} else if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
	}
	return subjects, nil
}

func (c Checker) enabledRules(ctx context.Context, subjects []Subject) ([]Rule, error) {
	where, args := pairPredicate("subject_type", "subject_id", subjects, 1)
	rows, err := c.DB.Query(ctx, `
		SELECT subject_type, subject_id, metric, max_value
		FROM usage_limits
		WHERE enabled AND (`+where+`)
		ORDER BY CASE subject_type
			WHEN 'organisation' THEN 1
			WHEN 'project' THEN 2
			WHEN 'lrs' THEN 3
			ELSE 4
		END, metric
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(&rule.SubjectType, &rule.SubjectID, &rule.Metric, &rule.Max); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (c Checker) loadUsage(ctx context.Context, subjects []Subject) (map[Subject]Usage, error) {
	out := make(map[Subject]Usage, len(subjects))
	for _, subject := range subjects {
		query := ""
		switch subject.Type {
		case SubjectOrganisation:
			query = `SELECT statement_count, storage_bytes FROM organisations WHERE id = $1::uuid`
		case SubjectProject:
			query = `SELECT statement_count, storage_bytes FROM projects WHERE id = $1::uuid`
		case SubjectLRS:
			query = `SELECT statement_count, storage_bytes FROM lrs WHERE id = $1`
		case SubjectClient:
			query = `SELECT statement_count, storage_bytes FROM clients WHERE id = $1::uuid`
		default:
			continue
		}
		var u Usage
		err := c.DB.QueryRow(ctx, query, subject.ID).Scan(&u.StatementsTotal, &u.StorageBytes)
		if err == pgx.ErrNoRows {
			out[subject] = Usage{}
			continue
		}
		if err != nil {
			return nil, err
		}
		out[subject] = u
	}

	if err := c.fillWindow(ctx, subjects, out, true); err != nil {
		return nil, err
	}
	if err := c.fillWindow(ctx, subjects, out, false); err != nil {
		return nil, err
	}
	return out, nil
}

func (c Checker) fillWindow(ctx context.Context, subjects []Subject, out map[Subject]Usage, hourly bool) error {
	where, args := pairPredicate("scope_type", "scope_id", subjects, 2)
	now := time.Now().UTC()
	var query string
	var window any
	if hourly {
		window = now.Truncate(time.Hour)
		query = `
			SELECT scope_type, scope_id, statement_count
			FROM statement_stats_hourly
			WHERE hour = $1 AND (` + where + `)`
	} else {
		window = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		query = `
			SELECT scope_type, scope_id, statement_count
			FROM statement_stats_daily
			WHERE day = $1 AND (` + where + `)`
	}
	rows, err := c.DB.Query(ctx, query, append([]any{window}, args...)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var typ, id string
		var n int64
		if err := rows.Scan(&typ, &id, &n); err != nil {
			return err
		}
		key := Subject{Type: typ, ID: id}
		u := out[key]
		if hourly {
			u.StatementsThisHour = n
		} else {
			u.StatementsToday = n
		}
		out[key] = u
	}
	return rows.Err()
}

// pairPredicate builds "(typeCol = $n AND idCol = $n+1) OR ...".
// startAt is the first placeholder index.
func pairPredicate(typeCol, idCol string, subjects []Subject, startAt int) (string, []any) {
	parts := make([]string, 0, len(subjects))
	args := make([]any, 0, len(subjects)*2)
	n := startAt
	for _, subject := range subjects {
		parts = append(parts, fmt.Sprintf("(%s = $%d AND %s = $%d)", typeCol, n, idCol, n+1))
		args = append(args, subject.Type, subject.ID)
		n += 2
	}
	if len(parts) == 0 {
		return "false", nil
	}
	return strings.Join(parts, " OR "), args
}
