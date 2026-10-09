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

const voidVerbID = "http://adlnet.gov/expapi/verbs/voided"

type mongoAgent struct {
	Mbox        string `bson:"mbox,omitempty"`
	MboxSHA1Sum string `bson:"mbox_sha1sum,omitempty"`
	OpenID      string `bson:"openid,omitempty"`
	AccountHome string `bson:"account_home,omitempty"`
	AccountName string `bson:"account_name,omitempty"`
}

type StatementStore struct {
	col *mongo.Collection
}

func NewStatementStore(db *mongo.Database) *StatementStore {
	return &StatementStore{col: db.Collection(colStatements)}
}

type statementDoc struct {
	// _id is composite "tenant_id|id" so idempotency does not need a second unique index.
	DocID         string       `bson:"_id"`
	TenantID      string       `bson:"tenant_id"`
	ID            string       `bson:"id"`
	StoredAt      time.Time    `bson:"stored_at"`
	Timestamp     *time.Time   `bson:"timestamp,omitempty"`
	ClientID      *string      `bson:"client_id,omitempty"`
	ActorMBox     *string      `bson:"actor_mbox,omitempty"`
	ActorSHA256   *string      `bson:"actor_sha256,omitempty"`
	VerbID        *string      `bson:"verb_id,omitempty"`
	ObjectID      *string      `bson:"object_id,omitempty"`
	Registration  *string      `bson:"registration,omitempty"`
	ObjectType    *string      `bson:"object_type,omitempty"`
	ActorMboxSHA  *string      `bson:"actor_mbox_sha1sum,omitempty"`
	ActorOpenID   *string      `bson:"actor_openid,omitempty"`
	ActorAcctHome *string      `bson:"actor_account_home,omitempty"`
	ActorAcctName *string      `bson:"actor_account_name,omitempty"`
	ActivityIDs   []string     `bson:"activity_ids,omitempty"`
	Voided        bool         `bson:"voided"`
	Agents        []mongoAgent `bson:"agents,omitempty"`
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
		DocID:         string(st.TenantID) + "|" + string(st.ID),
		TenantID:      string(st.TenantID),
		ID:            string(st.ID),
		StoredAt:      st.StoredAt,
		Timestamp:     st.Timestamp,
		ClientID:      st.ClientID,
		ActorMBox:     st.ActorMBox,
		ActorSHA256:   st.ActorSHA256,
		VerbID:        st.VerbID,
		ObjectID:      st.ObjectID,
		Registration:  st.Registration,
		ObjectType:    st.ObjectType,
		ActorMboxSHA:  st.ActorMboxSHA1Sum,
		ActorOpenID:   st.ActorOpenID,
		ActorAcctHome: st.ActorAccountHome,
		ActorAcctName: st.ActorAccountName,
		ActivityIDs:   st.ActivityIDs,
		Voided:        st.Voided,
		Agents:        mongoAgents(st, payload),
		RawJSON:       append([]byte(nil), st.RawJSON...),
		Raw:           encoded,
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
		ObjectType:   d.ObjectType,
		Voided:       d.Voided,
		RawJSON:      rawJSON,
	}, nil
}

func mongoAgents(st store.Statement, payload map[string]any) []mongoAgent {
	if len(st.Agents) > 0 {
		out := make([]mongoAgent, 0, len(st.Agents))
		for _, a := range st.Agents {
			out = append(out, mongoAgent{
				Mbox:        deref(a.Mbox),
				MboxSHA1Sum: deref(a.MboxSHA1Sum),
				OpenID:      deref(a.OpenID),
				AccountHome: deref(a.AccountHome),
				AccountName: deref(a.AccountName),
			})
		}
		return out
	}
	return agentsFromPayload(payload)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func agentsFromPayload(payload map[string]any) []mongoAgent {
	actor, _ := payload["actor"].(map[string]any)
	if actor == nil {
		return nil
	}
	var out []mongoAgent
	add := func(m map[string]any) {
		ag := mongoAgent{}
		if mb, ok := m["mbox"].(string); ok {
			ag.Mbox = mb
		}
		if home, ok := m["account"].(map[string]any); ok {
			ag.AccountName, _ = home["name"].(string)
			ag.AccountHome, _ = home["homePage"].(string)
		}
		if ag.Mbox != "" || ag.AccountHome != "" || ag.AccountName != "" {
			out = append(out, ag)
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
		if !store.StatementsEquivalent(got.RawJSON, st.RawJSON) {
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

func (s *StatementStore) MarkVoided(ctx context.Context, tenant store.TenantID, id store.StatementID) error {
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: string(tenant) + "|" + string(id)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "voided", Value: true}}}},
	)
	return err
}

func (s *StatementStore) HasVoidingStatement(ctx context.Context, tenant store.TenantID, id store.StatementID) (bool, error) {
	err := s.col.FindOne(ctx, bson.D{
		{Key: "tenant_id", Value: string(tenant)},
		{Key: "verb_id", Value: voidVerbID},
		{Key: "object_id", Value: string(id)},
		{Key: "object_type", Value: "StatementRef"},
		{Key: "voided", Value: bson.D{{Key: "$ne", Value: true}}},
	}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *StatementStore) GetStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit < 0 {
		f.Limit = 50
	}

	filter := bson.D{}
	if f.TenantID != "" {
		filter = append(filter, bson.E{Key: "tenant_id", Value: string(f.TenantID)})
	}
	switch {
	case f.VoidedStatementID != nil:
		filter = append(filter,
			bson.E{Key: "id", Value: *f.VoidedStatementID},
			bson.E{Key: "voided", Value: true},
		)
	case f.StatementID != nil:
		filter = append(filter,
			bson.E{Key: "id", Value: *f.StatementID},
			bson.E{Key: "voided", Value: bson.D{{Key: "$ne", Value: true}}},
		)
	default:
		if !f.IncludeVoided {
			filter = append(filter, bson.E{Key: "voided", Value: bson.D{{Key: "$ne", Value: true}}})
		}
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
		if f.RelatedActivities {
			filter = append(filter, bson.E{Key: "$or", Value: bson.A{
				bson.D{{Key: "object_id", Value: *f.ObjectID}},
				bson.D{{Key: "activity_ids", Value: *f.ObjectID}},
			}})
		} else {
			filter = append(filter, bson.E{Key: "object_id", Value: *f.ObjectID})
		}
	}
	if f.Registration != nil {
		filter = append(filter, bson.E{Key: "registration", Value: *f.Registration})
	}
	if f.ClientID != nil {
		filter = append(filter, bson.E{Key: "client_id", Value: *f.ClientID})
	}
	if f.Agent != nil {
		filter = append(filter, agentFilter(*f.Agent, f.RelatedAgents))
	}
	storedRange := bson.D{}
	if f.StoredSince != nil {
		storedRange = append(storedRange, bson.E{Key: "$gte", Value: *f.StoredSince})
	}
	if f.StoredUntil != nil {
		storedRange = append(storedRange, bson.E{Key: "$lte", Value: *f.StoredUntil})
	}
	if len(storedRange) > 0 {
		filter = append(filter, bson.E{Key: "stored_at", Value: storedRange})
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
	var query any = filter
	if ors := splitOrs(filter); ors != nil {
		query = bson.D{{Key: "$and", Value: ors}}
	}
	cursor, err := s.col.Find(ctx, query,
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
	if f.Limit > 0 && len(docs) > f.Limit {
		last := docs[f.Limit-1]
		token := store.EncodeCursor(store.Cursor{StoredAt: last.StoredAt, ID: store.StatementID(last.ID)})
		nextCursor = &token
		docs = docs[:f.Limit]
	} else if f.Limit == 0 {
		docs = nil
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

func splitOrs(filter bson.D) bson.A {
	n := 0
	for _, e := range filter {
		if e.Key == "$or" {
			n++
		}
	}
	if n < 2 {
		return nil
	}
	parts := bson.A{}
	rest := bson.D{}
	for _, e := range filter {
		if e.Key == "$or" {
			parts = append(parts, bson.D{e})
			continue
		}
		rest = append(rest, e)
	}
	if len(rest) > 0 {
		parts = append(bson.A{rest}, parts...)
	}
	return parts
}

func agentFilter(agent store.AgentRef, related bool) bson.E {
	match := bson.D{}
	if agent.Mbox != nil {
		match = append(match, bson.E{Key: "mbox", Value: *agent.Mbox})
	}
	if agent.MboxSHA1Sum != nil {
		match = append(match, bson.E{Key: "mbox_sha1sum", Value: *agent.MboxSHA1Sum})
	}
	if agent.OpenID != nil {
		match = append(match, bson.E{Key: "openid", Value: *agent.OpenID})
	}
	if agent.AccountHome != nil {
		match = append(match, bson.E{Key: "account_home", Value: *agent.AccountHome})
	}
	if agent.AccountName != nil {
		match = append(match, bson.E{Key: "account_name", Value: *agent.AccountName})
	}
	if related {
		return bson.E{Key: "agents", Value: bson.D{{Key: "$elemMatch", Value: match}}}
	}
	out := bson.D{}
	if agent.Mbox != nil {
		out = append(out, bson.E{Key: "actor_mbox", Value: *agent.Mbox})
	}
	if agent.MboxSHA1Sum != nil {
		out = append(out, bson.E{Key: "actor_mbox_sha1sum", Value: *agent.MboxSHA1Sum})
	}
	if agent.OpenID != nil {
		out = append(out, bson.E{Key: "actor_openid", Value: *agent.OpenID})
	}
	if agent.AccountHome != nil {
		out = append(out, bson.E{Key: "actor_account_home", Value: *agent.AccountHome})
	}
	if agent.AccountName != nil {
		out = append(out, bson.E{Key: "actor_account_name", Value: *agent.AccountName})
	}
	if len(out) == 0 {
		return bson.E{Key: "id", Value: bson.D{{Key: "$exists", Value: false}}}
	}
	if len(out) == 1 {
		return out[0]
	}
	return bson.E{Key: "$and", Value: []bson.D{out}}
}
