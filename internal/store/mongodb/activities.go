package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type ActivityStore struct {
	col *mongo.Collection
}

func NewActivityStore(db *mongo.Database) *ActivityStore {
	return &ActivityStore{col: db.Collection(colActivities)}
}

func (s *ActivityStore) PutActivity(ctx context.Context, tenant store.TenantID, id string, definition []byte) error {
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: string(tenant) + "|" + id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "tenant_id", Value: string(tenant)},
			{Key: "id", Value: id},
			{Key: "definition", Value: definition},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *ActivityStore) GetActivity(ctx context.Context, tenant store.TenantID, id string) ([]byte, bool, error) {
	var doc struct {
		Definition []byte `bson:"definition"`
	}
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: string(tenant) + "|" + id}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return doc.Definition, true, nil
}
