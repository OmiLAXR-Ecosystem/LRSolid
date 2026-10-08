package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type clientLifecycle struct {
	client *mongo.Client
}

func (c *clientLifecycle) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.client.Disconnect(ctx)
}

func (c *clientLifecycle) Health(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

// NewStoreBundle is a store.StoreFactory for MongoDB.
// The connection string must be a valid MongoDB URI, optionally with the
// database name as the path component: mongodb://host/dbname
// If no database name is found in the URI, "lrsolid" is used as default.
func NewStoreBundle(ctx context.Context, cfg store.DataStoreConfig) (store.StoreBundle, error) {
	dbName := dbNameFromURI(cfg.ConnectionString)
	client, db, err := Open(ctx, cfg.ConnectionString, dbName)
	if err != nil {
		return store.StoreBundle{}, fmt.Errorf("mongodb connect: %w", err)
	}

	return store.StoreBundle{
		Statements:       NewStatementStore(db),
		States:           NewStateStore(db),
		AgentProfiles:    NewAgentProfileStore(db),
		ActivityProfiles: NewActivityProfileStore(db),
		Attachments:      NewAttachmentStore(db),
		Lifecycle:        &clientLifecycle{client: client},
		DatabaseType:     "mongodb",
		Capabilities:     store.ExternalCapabilities(),
	}, nil
}

// dbNameFromURI extracts the database name from a MongoDB URI path.
// Falls back to "lrsolid" if not present.
func dbNameFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return "lrsolid"
	}
	if u.Path != "" && u.Path != "/" {
		name := u.Path[1:] // strip leading /
		if name != "" {
			return name
		}
	}
	return "lrsolid"
}
