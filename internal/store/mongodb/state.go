package mongodb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type StateStore struct {
	col *mongo.Collection
}

func NewStateStore(db *mongo.Database) *StateStore {
	return &StateStore{col: db.Collection(colStateDocs)}
}

type stateDoc struct {
	DocID        string    `bson:"_id"`
	TenantID     string    `bson:"tenant_id"`
	ActivityID   string    `bson:"activity_id"`
	AgentKey     string    `bson:"agent_key"`
	Registration *string   `bson:"registration,omitempty"`
	StateID      string    `bson:"state_id"`
	Content      []byte    `bson:"content"`
	ContentType  *string   `bson:"content_type,omitempty"`
	ETag         string    `bson:"etag"`
	UpdatedAt    time.Time `bson:"updated_at"`
}

func stateDocID(key store.StateKey) string {
	reg := ""
	if key.Registration != nil {
		reg = *key.Registration
	}
	return string(key.TenantID) + "|" + key.ActivityID + "|" + key.AgentKey + "|" + reg + "|" + key.StateID
}

func stateFilter(key store.StateKey) bson.D {
	return bson.D{{Key: "_id", Value: stateDocID(key)}}
}

func computeStateETag(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (s *StateStore) PutState(ctx context.Context, key store.StateKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	newETag := computeStateETag(content)
	now := time.Now().UTC()

	var ct *string
	if contentType != "" {
		ct = &contentType
	}

	doc := stateDoc{
		DocID:        stateDocID(key),
		TenantID:     string(key.TenantID),
		ActivityID:   key.ActivityID,
		AgentKey:     key.AgentKey,
		Registration: key.Registration,
		StateID:      key.StateID,
		Content:      content,
		ContentType:  ct,
		ETag:         newETag,
		UpdatedAt:    now,
	}

	if ifMatch != nil {
		// Conditional update: only if existing etag matches
		filter := bson.D{
			{Key: "_id", Value: doc.DocID},
			{Key: "etag", Value: string(*ifMatch)},
		}
		result, err := s.col.ReplaceOne(ctx, filter, doc)
		if err != nil {
			return "", err
		}
		if result.MatchedCount == 0 {
			return "", store.ErrPreconditionFailed
		}
		return store.ETag(newETag), nil
	}

	// Unconditional upsert
	_, err := s.col.ReplaceOne(ctx, stateFilter(key), doc, options.Replace().SetUpsert(true))
	if err != nil {
		return "", err
	}
	return store.ETag(newETag), nil
}

func (s *StateStore) GetState(ctx context.Context, key store.StateKey) (store.StateDoc, bool, error) {
	var doc stateDoc
	err := s.col.FindOne(ctx, stateFilter(key)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return store.StateDoc{}, false, nil
	}
	if err != nil {
		return store.StateDoc{}, false, err
	}

	out := store.StateDoc{
		Key:       key,
		Content:   doc.Content,
		ETag:      store.ETag(doc.ETag),
		UpdatedAt: doc.UpdatedAt,
	}
	if doc.ContentType != nil {
		out.ContentType = *doc.ContentType
	}
	return out, true, nil
}

func (s *StateStore) DeleteState(ctx context.Context, key store.StateKey) error {
	_, err := s.col.DeleteOne(ctx, stateFilter(key))
	return err
}

func (s *StateStore) ListStateIDs(ctx context.Context, tenant store.TenantID, activityID, agentKey string, registration *string, since *time.Time) ([]string, error) {
	filter := bson.D{
		{Key: "tenant_id", Value: string(tenant)},
		{Key: "activity_id", Value: activityID},
		{Key: "agent_key", Value: agentKey},
	}
	if registration != nil {
		filter = append(filter, bson.E{Key: "registration", Value: *registration})
	} else {
		filter = append(filter, bson.E{Key: "registration", Value: bson.D{{Key: "$exists", Value: false}}})
	}
	if since != nil {
		filter = append(filter, bson.E{Key: "updated_at", Value: bson.D{{Key: "$gt", Value: *since}}})
	}

	cursor, err := s.col.Find(ctx, filter,
		options.Find().SetProjection(bson.D{{Key: "state_id", Value: 1}}).SetSort(bson.D{{Key: "state_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var out []string
	for cursor.Next(ctx) {
		var doc struct {
			StateID string `bson:"state_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.StateID)
	}
	return out, cursor.Err()
}
