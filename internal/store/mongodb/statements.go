package mongodb

import (
	"context"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type StatementStore struct {
	col *mongo.Collection
}

func NewStatementStore(db *mongo.Database) *StatementStore {
	return &StatementStore{col: db.Collection(colStatements)}
}

type statementDoc struct {
	// _id is composite "tenant_id|id" so idempotency does not need a second unique index.
	DocID        string     `bson:"_id"`
	TenantID     string     `bson:"tenant_id"`
	ID           string     `bson:"id"`
	StoredAt     time.Time  `bson:"stored_at"`
	Timestamp    *time.Time `bson:"timestamp,omitempty"`
	ClientID     *string    `bson:"client_id,omitempty"`
	ActorMBox    *string    `bson:"actor_mbox,omitempty"`
	ActorSHA256  *string    `bson:"actor_sha256,omitempty"`
	VerbID       *string    `bson:"verb_id,omitempty"`
	ObjectID     *string    `bson:"object_id,omitempty"`
	Registration *string    `bson:"registration,omitempty"`
	Agents       []string   `bson:"agents,omitempty"`
	// RawJSON is the original xAPI document, used for idempotent replays.
	RawJSON []byte `bson:"raw_json,omitempty"`
	// Raw is the same document with dotted extension keys escaped so MongoDB can store and index it.
	Raw bson.Raw `bson:"raw,omitempty"`
}

func toDoc(st store.Statement) (statementDoc, error) {
	var payload map[string]any
	if err := json.Unmarshal(st.RawJSON, &payload); err != nil {
		return statementDoc{}, err
	}
	encoded, err := bson.Marshal(escapeValue(payload))
	if err != nil {
		return statementDoc{}, err
	}
	return statementDoc{
		DocID:        string(st.TenantID) + "|" + string(st.ID),
		TenantID:     string(st.TenantID),
		ID:           string(st.ID),
		StoredAt:     st.StoredAt,
		Timestamp:    st.Timestamp,
		ClientID:     st.ClientID,
		ActorMBox:    st.ActorMBox,
		ActorSHA256:  st.ActorSHA256,
		VerbID:       st.VerbID,
		ObjectID:     st.ObjectID,
		Registration: st.Registration,
		Agents:       agentIFIDs(payload),
		RawJSON:      append([]byte(nil), st.RawJSON...),
		Raw:          encoded,
	}, nil
}

func fromDoc(d statementDoc) (store.Statement, error) {
	rawJSON := d.RawJSON
	if len(rawJSON) == 0 && len(d.Raw) > 0 {
		var escaped any
		if err := bson.Unmarshal(d.Raw, &escaped); err != nil {
			return store.Statement{}, err
		}
		encoded, err := json.Marshal(unescapeValue(escaped))
		if err != nil {
			return store.Statement{}, err
		}
		rawJSON = encoded
	}
	return store.Statement{
		TenantID:     store.TenantID(d.TenantID),
		ID:           store.StatementID(d.ID),
		StoredAt:     d.StoredAt,
		Timestamp:    d.Timestamp,
		ClientID:     d.ClientID,
		ActorMBox:    d.ActorMBox,
		ActorSHA256:  d.ActorSHA256,
		VerbID:       d.VerbID,
		ObjectID:     d.ObjectID,
		Registration: d.Registration,
		RawJSON:      rawJSON,
	}, nil
}

func agentIFIDs(payload map[string]any) []string {
	actor, _ := payload["actor"].(map[string]any)
	if actor == nil {
		return nil
	}
	var out []string
	add := func(m map[string]any) {
		if mb, ok := m["mbox"].(string); ok && mb != "" {
			out = append(out, "mbox:"+mb)
		}
		if home, ok := m["account"].(map[string]any); ok {
			name, _ := home["name"].(string)
			page, _ := home["homePage"].(string)
			if name != "" || page != "" {
				out = append(out, "account:"+page+"|"+name)
			}
		}
	}
	add(actor)
	members, _ := actor["member"].([]any)
	for _, member := range members {
		if m, ok := member.(map[string]any); ok {
			add(m)
		}
	}
	return out
}

func (s *StatementStore) UpsertStatements(ctx context.Context, stmts []store.Statement) (int, error) {
	if len(stmts) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, st := range stmts {
		doc, err := toDoc(st)
		if err != nil {
			return inserted, err
		}
		_, err = s.col.InsertOne(ctx, doc)
		if err == nil {
			inserted++
			continue
		}
		if !mongo.IsDuplicateKeyError(err) {
			return inserted, err
		}
		var existing statementDoc
		if err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: doc.DocID}}).Decode(&existing); err != nil {
			return inserted, err
		}
		got, err := fromDoc(existing)
		if err != nil {
			return inserted, err
		}
		if !store.JSONEqual(got.RawJSON, st.RawJSON) {
			return inserted, store.ErrAlreadyExists
		}
	}
	return inserted, nil
}

func (s *StatementStore) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	var filter bson.D
	if tenant == "" {
		filter = bson.D{{Key: "id", Value: string(id)}}
	} else {
		filter = bson.D{{Key: "_id", Value: string(tenant) + "|" + string(id)}}
	}

	var doc statementDoc
	err := s.col.FindOne(ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return store.Statement{}, store.ErrNotFound
	}
	if err != nil {
		return store.Statement{}, err
	}
	return fromDoc(doc)
}

func (s *StatementStore) GetStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	filter := bson.D{}
	if f.TenantID != "" {
		filter = append(filter, bson.E{Key: "tenant_id", Value: string(f.TenantID)})
	}
	if f.ActorMBox != nil {
		filter = append(filter, bson.E{Key: "actor_mbox", Value: *f.ActorMBox})
	}
	if f.ActorSHA256 != nil {
		filter = append(filter, bson.E{Key: "actor_sha256", Value: *f.ActorSHA256})
	}
	if f.VerbID != nil {
		filter = append(filter, bson.E{Key: "verb_id", Value: *f.VerbID})
	}
	if f.ObjectID != nil {
		filter = append(filter, bson.E{Key: "object_id", Value: *f.ObjectID})
	}
	if f.Registration != nil {
		filter = append(filter, bson.E{Key: "registration", Value: *f.Registration})
	}
	if f.StoredSince != nil {
		filter = append(filter, bson.E{Key: "stored_at", Value: bson.D{{Key: "$gte", Value: *f.StoredSince}}})
	}
	if f.StoredUntil != nil {
		filter = append(filter, bson.E{Key: "stored_at", Value: bson.D{{Key: "$lte", Value: *f.StoredUntil}}})
	}

	// Cursor pagination: replicate (stored_at, id) tuple comparison
	if f.Cursor != nil {
		c, err := store.DecodeCursor(*f.Cursor)
		if err != nil {
			return store.StatementPage{}, err
		}
		var op string
		if f.Ascending {
			op = "$gt"
		} else {
			op = "$lt"
		}
		// Equivalent to SQL: (stored_at, id) > (c.StoredAt, c.ID)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "stored_at", Value: bson.D{{Key: op, Value: c.StoredAt}}}},
			bson.D{
				{Key: "stored_at", Value: c.StoredAt},
				{Key: "id", Value: bson.D{{Key: op, Value: string(c.ID)}}},
			},
		}})
	}

	sortDir := -1
	if f.Ascending {
		sortDir = 1
	}
	sort := bson.D{{Key: "stored_at", Value: sortDir}, {Key: "id", Value: sortDir}}

	limit := int64(f.Limit + 1)
	cursor, err := s.col.Find(ctx, filter,
		options.Find().SetSort(sort).SetLimit(limit),
	)
	if err != nil {
		return store.StatementPage{}, err
	}
	defer cursor.Close(ctx)

	var docs []statementDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return store.StatementPage{}, err
	}

	var nextCursor *string
	if len(docs) > f.Limit {
		last := docs[f.Limit-1]
		token := store.EncodeCursor(store.Cursor{StoredAt: last.StoredAt, ID: store.StatementID(last.ID)})
		nextCursor = &token
		docs = docs[:f.Limit]
	}

	stmts := make([]store.Statement, 0, len(docs))
	for _, d := range docs {
		st, err := fromDoc(d)
		if err != nil {
			return store.StatementPage{}, err
		}
		stmts = append(stmts, st)
	}

	return store.StatementPage{Statements: stmts, NextCursor: nextCursor}, nil
}
