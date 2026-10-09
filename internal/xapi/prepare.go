package xapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SGoerzen/LRSolid/internal/store"
)

// ActivityDef is a definition observed on a statement, used to update the canonical activity.
type ActivityDef struct {
	ID         string
	Definition json.RawMessage
}

// Prepared is a validated statement plus the columns the stores index.
type Prepared struct {
	Statement  store.Statement
	Activities []ActivityDef
}

// Prepare validates one statement, assigns LRS properties, and extracts index fields.
// statementID, when set, is the id from a PUT query string.
func Prepare(raw json.RawMessage, version string, now time.Time, authorityName, authorityHome, statementID string) (Prepared, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return Prepared{}, invalid("invalid statement json")
	}
	if err := ValidateStatement(m, version, now); err != nil {
		return Prepared{}, err
	}

	id := ""
	if v, ok := m["id"].(string); ok && v != "" {
		id = v
	}
	if statementID != "" {
		if id != "" && !strings.EqualFold(id, statementID) {
			return Prepared{}, invalid("statement id does not match statementId")
		}
		id = statementID
	}
	if id == "" {
		id = uuid.NewString()
	}
	m["id"] = id

	stamp := formatStamp(now)
	m["stored"] = stamp
	if _, ok := m["timestamp"]; !ok {
		m["timestamp"] = stamp
	}
	if version == "" {
		version = Version103
	}
	m["version"] = version
	if authorityName == "" {
		authorityName = "lrs"
	}
	if authorityHome == "" {
		authorityHome = "http://localhost"
	}
	m["authority"] = map[string]any{
		"objectType": "Agent",
		"account": map[string]any{
			"homePage": authorityHome,
			"name":     authorityName,
		},
	}
	lowercaseMbox(m)

	body, err := marshal(m)
	if err != nil {
		return Prepared{}, err
	}
	st := extract(m, body, now)
	return Prepared{Statement: st, Activities: activityDefs(m)}, nil
}

func formatStamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func lowercaseMbox(v any) {
	switch n := v.(type) {
	case map[string]any:
		if s, ok := n["mbox"].(string); ok {
			n["mbox"] = strings.ToLower(s)
		}
		for _, child := range n {
			lowercaseMbox(child)
		}
	case []any:
		for _, child := range n {
			lowercaseMbox(child)
		}
	}
}

func extract(m map[string]any, raw []byte, stored time.Time) store.Statement {
	st := store.Statement{
		ID:       store.StatementID(m["id"].(string)),
		StoredAt: stored.UTC(),
		RawJSON:  raw,
	}
	if ts, ok := m["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			u := t.UTC()
			st.Timestamp = &u
		}
	}
	if actor, ok := m["actor"].(map[string]any); ok {
		ot, _ := actor["objectType"].(string)
		if ot == "" {
			ot = "Agent"
		}
		st.ActorType = &ot
		applyIFI(&st, actor)
		if ref, ok := ifiOf(actor); ok && ref.Mbox != nil {
			sum := store.HashMBox(*ref.Mbox)
			st.ActorSHA256 = &sum
		}
	}
	if verb, ok := m["verb"].(map[string]any); ok {
		if id, ok := verb["id"].(string); ok && id != "" {
			st.VerbID = &id
		}
	}
	if obj, ok := m["object"].(map[string]any); ok {
		typ, id, actType := objectInfo(obj)
		if typ != "" {
			st.ObjectType = &typ
		}
		if id != "" {
			st.ObjectID = &id
		}
		if actType != "" {
			st.ActivityType = &actType
		}
		if typ == "StatementRef" && st.VerbID != nil && *st.VerbID == VerbVoided {
			target := store.StatementID(id)
			st.VoidsID = &target
		}
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		if reg, ok := ctx["registration"].(string); ok && reg != "" {
			st.Registration = &reg
		}
		if p, ok := ctx["platform"].(string); ok && p != "" {
			st.ContextPlatform = &p
		}
		if ca, ok := ctx["contextActivities"].(map[string]any); ok {
			st.ContextParentIDs = activityListIDs(ca["parent"])
			st.ContextGroupIDs = activityListIDs(ca["grouping"])
			st.ContextCategoryIDs = activityListIDs(ca["category"])
		}
	}
	if res, ok := m["result"].(map[string]any); ok {
		if v, ok := res["completion"].(bool); ok {
			st.ResultCompletion = &v
		}
		if v, ok := res["success"].(bool); ok {
			st.ResultSuccess = &v
		}
		if d, ok := res["duration"].(string); ok {
			st.ResultDuration = &d
		}
		if score, ok := res["score"].(map[string]any); ok {
			if n, ok := asFloat(score["scaled"]); ok {
				st.ResultScoreScaled = &n
			}
		}
	}
	st.ActivityIDs = uniqueStrings(collectActivityIDs(m))
	st.Agents = collectAgents(m)
	return st
}

func applyIFI(st *store.Statement, actor map[string]any) {
	ref, ok := ifiOf(actor)
	if !ok {
		return
	}
	st.ActorMBox = ref.Mbox
	st.ActorMboxSHA1Sum = ref.MboxSHA1Sum
	st.ActorOpenID = ref.OpenID
	st.ActorAccountHome = ref.AccountHome
	st.ActorAccountName = ref.AccountName
}

func objectInfo(obj map[string]any) (typ, id, activityType string) {
	ot, _ := obj["objectType"].(string)
	switch ot {
	case "", "Activity":
		typ = "Activity"
		id, _ = obj["id"].(string)
		if def, ok := obj["definition"].(map[string]any); ok {
			activityType, _ = def["type"].(string)
		}
	case "StatementRef":
		typ = "StatementRef"
		id, _ = obj["id"].(string)
	case "Agent", "Group":
		typ = ot
	case "SubStatement":
		typ = "SubStatement"
		if inner, ok := obj["object"].(map[string]any); ok {
			_, id, activityType = objectInfo(inner)
		}
	default:
		typ = ot
		id, _ = obj["id"].(string)
	}
	return typ, id, activityType
}

func collectActivityIDs(m map[string]any) []string {
	var out []string
	if obj, ok := m["object"].(map[string]any); ok {
		out = append(out, activityIDsFromObject(obj)...)
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		if ca, ok := ctx["contextActivities"].(map[string]any); ok {
			for _, key := range []string{"parent", "grouping", "category", "other"} {
				out = append(out, activityListIDs(ca[key])...)
			}
		}
	}
	return out
}

func activityIDsFromObject(obj map[string]any) []string {
	ot, _ := obj["objectType"].(string)
	switch ot {
	case "", "Activity":
		if id, ok := obj["id"].(string); ok && id != "" {
			return []string{id}
		}
	case "SubStatement":
		return collectActivityIDs(obj)
	}
	return nil
}

func activityListIDs(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["id"].(string); ok && id != "" {
			out = append(out, id)
		}
	}
	return out
}

func collectAgents(m map[string]any) []store.AgentRef {
	var out []store.AgentRef
	add := func(agent map[string]any) {
		if ref, ok := ifiOf(agent); ok {
			out = append(out, ref)
		}
		if members, ok := agent["member"].([]any); ok {
			for _, member := range members {
				if mm, ok := member.(map[string]any); ok {
					if ref, ok := ifiOf(mm); ok {
						out = append(out, ref)
					}
				}
			}
		}
	}
	if actor, ok := m["actor"].(map[string]any); ok {
		add(actor)
	}
	if obj, ok := m["object"].(map[string]any); ok {
		ot, _ := obj["objectType"].(string)
		switch ot {
		case "Agent", "Group":
			add(obj)
		case "SubStatement":
			out = append(out, collectAgents(obj)...)
		}
	}
	ctx, _ := m["context"].(map[string]any)
	if ctx != nil {
		if inst, ok := ctx["instructor"].(map[string]any); ok {
			add(inst)
		}
		if team, ok := ctx["team"].(map[string]any); ok {
			add(team)
		}
		if arr, ok := ctx["contextAgents"].([]any); ok {
			for _, item := range arr {
				cm, _ := item.(map[string]any)
				if agent, ok := cm["agent"].(map[string]any); ok {
					add(agent)
				}
			}
		}
		if arr, ok := ctx["contextGroups"].([]any); ok {
			for _, item := range arr {
				cm, _ := item.(map[string]any)
				if group, ok := cm["group"].(map[string]any); ok {
					add(group)
				}
			}
		}
	}
	return out
}

func activityDefs(m map[string]any) []ActivityDef {
	var out []ActivityDef
	var walk func(obj map[string]any)
	walk = func(obj map[string]any) {
		ot, _ := obj["objectType"].(string)
		if ot == "" || ot == "Activity" {
			id, _ := obj["id"].(string)
			if def, ok := obj["definition"].(map[string]any); ok && id != "" {
				if raw, err := marshal(def); err == nil {
					out = append(out, ActivityDef{ID: id, Definition: raw})
				}
			}
		}
		if ot == "SubStatement" {
			if inner, ok := obj["object"].(map[string]any); ok {
				walk(inner)
			}
			if ctx, ok := obj["context"].(map[string]any); ok {
				collectContextDefs(ctx, &out)
			}
		}
	}
	if obj, ok := m["object"].(map[string]any); ok {
		walk(obj)
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		collectContextDefs(ctx, &out)
	}
	return out
}

func collectContextDefs(ctx map[string]any, out *[]ActivityDef) {
	ca, _ := ctx["contextActivities"].(map[string]any)
	if ca == nil {
		return
	}
	for _, key := range []string{"parent", "grouping", "category", "other"} {
		arr, _ := ca[key].([]any)
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id, _ := m["id"].(string)
			def, ok := m["definition"].(map[string]any)
			if !ok || id == "" {
				continue
			}
			if raw, err := marshal(def); err == nil {
				*out = append(*out, ActivityDef{ID: id, Definition: raw})
			}
		}
	}
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// AttachmentPart is one multipart body hashed by SHA-256.
type AttachmentPart struct {
	SHA2        string
	ContentType string
	Body        []byte
}

// CheckAttachments verifies sha2, length, and that unsigned parts were supplied
// unless the descriptor only references a fileUrl. Signature parts must be present.
func CheckAttachments(statement map[string]any, parts map[string]AttachmentPart) error {
	atts, ok := statement["attachments"].([]any)
	if !ok {
		if len(parts) > 0 {
			return invalid("multipart parts do not match attachments")
		}
		return nil
	}
	seen := map[string]bool{}
	for _, item := range atts {
		m, _ := item.(map[string]any)
		sha, _ := m["sha2"].(string)
		sha = strings.ToLower(sha)
		seen[sha] = true
		length, _ := asFloat(m["length"])
		usage, _ := m["usageType"].(string)
		_, hasURL := m["fileUrl"].(string)
		part, ok := parts[sha]
		if !ok {
			if usage == AttachmentSignature || !hasURL {
				return invalid("missing attachment content for %s", sha)
			}
			continue
		}
		if int(length) != len(part.Body) {
			return invalid("attachment length does not match content")
		}
		sum := sha256Hex(part.Body)
		if sum != sha {
			return invalid("attachment sha2 does not match content")
		}
	}
	for sha := range parts {
		if !seen[sha] {
			return invalid("unexpected attachment part %s", sha)
		}
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256Sum(b)
	return fmt.Sprintf("%x", sum)
}
