package datastore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/store"
)

// LoadActive reads external data stores that should be connected.
// The seeded control-plane row has a host in connection_string rather than a URI;
// the registry ignores those.
func LoadActive(ctx context.Context, db *pgxpool.Pool, dataSecrets secrets.Store) ([]store.DataStoreConfig, error) {
	rows, err := db.Query(ctx, `
		SELECT id::text, name, database_type, connection_string, secret_ref
		FROM data_stores
		WHERE is_active = true
		  AND (
		    (connection_string IS NOT NULL AND connection_string != '')
		    OR (secret_ref IS NOT NULL AND secret_ref != '')
		  )
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.DataStoreConfig
	for rows.Next() {
		var cfg store.DataStoreConfig
		var connectionString, secretRef *string
		if err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.DatabaseType, &connectionString, &secretRef); err != nil {
			return nil, err
		}
		switch {
		case secretRef != nil && *secretRef != "":
			if dataSecrets == nil {
				continue
			}
			plaintext, err := dataSecrets.Get(ctx, *secretRef)
			if err != nil {
				continue
			}
			cfg.ConnectionString = string(plaintext)
		case connectionString != nil:
			cfg.ConnectionString = *connectionString
		}
		if cfg.ConnectionString == "" {
			continue
		}
		out = append(out, cfg)
	}
	return out, rows.Err()
}
