package limits

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ApplyOrgDefaults inserts organisation usage rules for each positive limit.
// Existing rules are left unchanged, so a later manual edit in the limits UI
// is not overwritten when the same organisation is created again.
func ApplyOrgDefaults(ctx context.Context, tx pgx.Tx, orgID string, storageBytes, perDay int64) error {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil
	}
	rules := []struct {
		metric string
		max    int64
	}{
		{MetricStorageBytes, storageBytes},
		{MetricStatementsPerDay, perDay},
	}
	for _, rule := range rules {
		if rule.max <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO usage_limits (subject_type, subject_id, metric, max_value, enabled)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (subject_type, subject_id, metric) DO NOTHING
		`, SubjectOrganisation, orgID, rule.metric, rule.max); err != nil {
			return err
		}
	}
	return nil
}
