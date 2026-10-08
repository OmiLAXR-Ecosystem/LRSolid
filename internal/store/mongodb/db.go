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

// Open connects to MongoDB and returns the client plus the selected database.
func Open(ctx context.Context, uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second).
		SetMaxPoolSize(20)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}

	db := client.Database(dbName)
	if err := ensureIndexes(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}
	return client, db, nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	// statements: primary query pattern + cursor pagination
	// Recommended shard key for horizontal scaling: { tenant_id: 1, stored_at: 1 }
	stmtCol := db.Collection(colStatements)
	// Older builds indexed the filter column without the sort key, which forces an in-memory sort.
	for _, legacy := range []string{
		"tenant_id_1_actor_mbox_1",
		"tenant_id_1_verb_id_1",
		"tenant_id_1_object_id_1",
		"tenant_id_1_registration_1",
	} {
		_ = stmtCol.Indexes().DropOne(ctx, legacy)
	}
	stored := bson.D{{Key: "stored_at", Value: -1}, {Key: "id", Value: -1}}
	stmtIndexes := []mongo.IndexModel{
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "actor_mbox", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "actor_sha256", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "verb_id", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "object_id", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "registration", Value: 1}}, stored...)},
		{Keys: append(bson.D{{Key: "tenant_id", Value: 1}, {Key: "agents", Value: 1}}, stored...)},
	}
	if _, err := stmtCol.Indexes().CreateMany(ctx, stmtIndexes); err != nil {
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
