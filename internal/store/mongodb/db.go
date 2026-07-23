package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	colStatements       = "statements"
	colStateDocs        = "state_docs"
	colAgentProfiles    = "agent_profiles"
	colActivityProfiles = "activity_profiles"
	colAttachmentMeta   = "attachment_meta"
)

// Open connects to MongoDB and returns a *mongo.Database.
func Open(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	db := client.Database(dbName)
	if err := ensureIndexes(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return db, nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	// statements: primary query pattern + cursor pagination
	// Recommended shard key for horizontal scaling: { tenant_id: 1, stored_at: 1 }
	stmtIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "stored_at", Value: -1}, {Key: "id", Value: -1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "actor_mbox", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "verb_id", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "object_id", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "registration", Value: 1}}},
	}
	if _, err := db.Collection(colStatements).Indexes().CreateMany(ctx, stmtIndexes); err != nil {
		return err
	}

	stateIndexes := []mongo.IndexModel{
		{Keys: bson.D{
			{Key: "tenant_id", Value: 1}, {Key: "activity_id", Value: 1},
			{Key: "agent_key", Value: 1}, {Key: "registration", Value: 1}, {Key: "state_id", Value: 1},
		}, Options: options.Index().SetUnique(true)},
	}
	if _, err := db.Collection(colStateDocs).Indexes().CreateMany(ctx, stateIndexes); err != nil {
		return err
	}

	agentIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "agent_key", Value: 1}, {Key: "profile_id", Value: 1}},
			Options: options.Index().SetUnique(true)},
	}
	if _, err := db.Collection(colAgentProfiles).Indexes().CreateMany(ctx, agentIdx); err != nil {
		return err
	}

	actIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "activity_id", Value: 1}, {Key: "profile_id", Value: 1}},
			Options: options.Index().SetUnique(true)},
	}
	if _, err := db.Collection(colActivityProfiles).Indexes().CreateMany(ctx, actIdx); err != nil {
		return err
	}

	attIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "sha256", Value: 1}},
			Options: options.Index().SetUnique(true)},
	}
	_, err := db.Collection(colAttachmentMeta).Indexes().CreateMany(ctx, attIdx)
	return err
}
