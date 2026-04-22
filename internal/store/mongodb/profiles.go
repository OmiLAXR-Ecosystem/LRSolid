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

type AgentProfileStore struct{ col *mongo.Collection }
type ActivityProfileStore struct{ col *mongo.Collection }

func NewAgentProfileStore(db *mongo.Database) *AgentProfileStore {
	return &AgentProfileStore{col: db.Collection(colAgentProfiles)}
}
func NewActivityProfileStore(db *mongo.Database) *ActivityProfileStore {
	return &ActivityProfileStore{col: db.Collection(colActivityProfiles)}
}

type profileDoc struct {
	DocID       string    `bson:"_id"`
	TenantID    string    `bson:"tenant_id"`
	KeyField    string    `bson:"key_field"` // agent_key or activity_id
	ProfileID   string    `bson:"profile_id"`
	Content     []byte    `bson:"content"`
	ContentType *string   `bson:"content_type,omitempty"`
	ETag        string    `bson:"etag"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

func profileETag(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func putProfile(ctx context.Context, col *mongo.Collection, docID, tenantID, keyField, profileID string, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	newETag := profileETag(content)
	now := time.Now().UTC()

	var ct *string
	if contentType != "" {
		ct = &contentType
	}

	doc := profileDoc{
		DocID:       docID,
		TenantID:    tenantID,
		KeyField:    keyField,
		ProfileID:   profileID,
		Content:     content,
		ContentType: ct,
		ETag:        newETag,
		UpdatedAt:   now,
	}

	if ifMatch != nil {
		filter := bson.D{{Key: "_id", Value: docID}, {Key: "etag", Value: string(*ifMatch)}}
		result, err := col.ReplaceOne(ctx, filter, doc)
		if err != nil {
			return "", err
		}
		if result.MatchedCount == 0 {
			return "", store.ErrPreconditionFailed
		}
		return store.ETag(newETag), nil
	}

	_, err := col.ReplaceOne(ctx, bson.D{{Key: "_id", Value: docID}}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return "", err
	}
	return store.ETag(newETag), nil
}

func getProfile(ctx context.Context, col *mongo.Collection, docID string) (store.ProfileDoc, bool, error) {
	var doc profileDoc
	err := col.FindOne(ctx, bson.D{{Key: "_id", Value: docID}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return store.ProfileDoc{}, false, nil
	}
	if err != nil {
		return store.ProfileDoc{}, false, err
	}
	out := store.ProfileDoc{
		Content:   doc.Content,
		ETag:      store.ETag(doc.ETag),
		UpdatedAt: doc.UpdatedAt,
	}
	if doc.ContentType != nil {
		out.ContentType = *doc.ContentType
	}
	return out, true, nil
}

func deleteProfile(ctx context.Context, col *mongo.Collection, docID string) error {
	_, err := col.DeleteOne(ctx, bson.D{{Key: "_id", Value: docID}})
	return err
}

func listProfileIDs(ctx context.Context, col *mongo.Collection, tenantID, keyFieldName, keyValue string, since *time.Time) ([]string, error) {
	filter := bson.D{
		{Key: "tenant_id", Value: tenantID},
		{Key: "key_field", Value: keyValue},
	}
	if since != nil {
		filter = append(filter, bson.E{Key: "updated_at", Value: bson.D{{Key: "$gt", Value: *since}}})
	}
	_ = keyFieldName

	cursor, err := col.Find(ctx, filter,
		options.Find().SetProjection(bson.D{{Key: "profile_id", Value: 1}}).SetSort(bson.D{{Key: "profile_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var out []string
	for cursor.Next(ctx) {
		var doc struct {
			ProfileID string `bson:"profile_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.ProfileID)
	}
	return out, cursor.Err()
}

// ---- AgentProfileStore ----

func (s *AgentProfileStore) PutAgentProfile(ctx context.Context, key store.AgentProfileKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	docID := string(key.TenantID) + "|" + key.AgentKey + "|" + key.ProfileID
	return putProfile(ctx, s.col, docID, string(key.TenantID), key.AgentKey, key.ProfileID, content, contentType, ifMatch)
}

func (s *AgentProfileStore) GetAgentProfile(ctx context.Context, key store.AgentProfileKey) (store.ProfileDoc, bool, error) {
	docID := string(key.TenantID) + "|" + key.AgentKey + "|" + key.ProfileID
	return getProfile(ctx, s.col, docID)
}

func (s *AgentProfileStore) DeleteAgentProfile(ctx context.Context, key store.AgentProfileKey) error {
	docID := string(key.TenantID) + "|" + key.AgentKey + "|" + key.ProfileID
	return deleteProfile(ctx, s.col, docID)
}

func (s *AgentProfileStore) ListAgentProfileIDs(ctx context.Context, tenant store.TenantID, agentKey string, since *time.Time) ([]string, error) {
	return listProfileIDs(ctx, s.col, string(tenant), "agent_key", agentKey, since)
}

// ---- ActivityProfileStore ----

func (s *ActivityProfileStore) PutActivityProfile(ctx context.Context, key store.ActivityProfileKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	docID := string(key.TenantID) + "|" + key.ActivityID + "|" + key.ProfileID
	return putProfile(ctx, s.col, docID, string(key.TenantID), key.ActivityID, key.ProfileID, content, contentType, ifMatch)
}

func (s *ActivityProfileStore) GetActivityProfile(ctx context.Context, key store.ActivityProfileKey) (store.ProfileDoc, bool, error) {
	docID := string(key.TenantID) + "|" + key.ActivityID + "|" + key.ProfileID
	return getProfile(ctx, s.col, docID)
}

func (s *ActivityProfileStore) DeleteActivityProfile(ctx context.Context, key store.ActivityProfileKey) error {
	docID := string(key.TenantID) + "|" + key.ActivityID + "|" + key.ProfileID
	return deleteProfile(ctx, s.col, docID)
}

func (s *ActivityProfileStore) ListActivityProfileIDs(ctx context.Context, tenant store.TenantID, activityID string, since *time.Time) ([]string, error) {
	return listProfileIDs(ctx, s.col, string(tenant), "activity_id", activityID, since)
}
