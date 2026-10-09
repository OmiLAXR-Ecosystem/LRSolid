package xapi

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string { return e.Reason }

func invalid(format string, args ...any) error {
	return &ValidationError{Reason: fmt.Sprintf(format, args...)}
}

var (
	uuidRe     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	sha1Re     = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	sha2Re     = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	durationRe = regexp.MustCompile(`^P(\d+Y)?(\d+M)?(\d+W)?(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`)
	versionRe  = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

var statementKeys = map[string]bool{
	"id": true, "actor": true, "verb": true, "object": true, "result": true,
	"context": true, "timestamp": true, "stored": true, "authority": true,
	"version": true, "attachments": true,
}

var subStatementKeys = map[string]bool{
	"objectType": true, "actor": true, "verb": true, "object": true,
	"result": true, "context": true, "timestamp": true, "attachments": true,
}

var interactionTypes = map[string]bool{
	"true-false": true, "choice": true, "fill-in": true, "long-fill-in": true,
	"matching": true, "performance": true, "sequencing": true, "likert": true,
	"numeric": true, "other": true,
}

// ValidateStatement checks a decoded statement against the negotiated xAPI version.
func ValidateStatement(m map[string]any, version string, now time.Time) error {
	return validateStatement(m, version, now, false)
}

func validateStatement(m map[string]any, version string, now time.Time, sub bool) error {
	allowed := statementKeys
	if sub {
		allowed = subStatementKeys
		if ot, _ := m["objectType"].(string); ot != "SubStatement" {
			return invalid("SubStatement objectType must be SubStatement")
		}
	}
	for k := range m {
		if !allowed[k] {
			return invalid("unknown property %q", k)
		}
	}
	if sub {
		for _, banned := range []string{"id", "stored", "version", "authority"} {
			if _, ok := m[banned]; ok {
				return invalid("SubStatement must not contain %s", banned)
			}
		}
	}

	actor, ok := m["actor"].(map[string]any)
	if !ok {
		return invalid("actor is required")
	}
	if err := validateAgent(actor, "actor", true); err != nil {
		return err
	}

	verb, ok := m["verb"].(map[string]any)
	if !ok {
		return invalid("verb is required")
	}
	if err := validateVerb(verb); err != nil {
		return err
	}

	obj, ok := m["object"].(map[string]any)
	if !ok {
		return invalid("object is required")
	}
	if err := validateObject(obj, version, now); err != nil {
		return err
	}

	if id, ok := m["id"]; ok {
		s, ok := id.(string)
		if !ok || !uuidRe.MatchString(s) {
			return invalid("id must be a UUID")
		}
	}
	if ts, ok := m["timestamp"]; ok {
		t, err := parseStamp(ts)
		if err != nil {
			return invalid("timestamp must be an RFC3339 timestamp with a timezone")
		}
		if Major(version) == "2" && t.After(now.Add(5*time.Second)) {
			return invalid("timestamp must not be in the future")
		}
	}
	if st, ok := m["stored"]; ok {
		if _, err := parseStamp(st); err != nil {
			return invalid("stored must be an RFC3339 timestamp with a timezone")
		}
	}
	if v, ok := m["version"]; ok {
		s, ok := v.(string)
		if !ok || !versionRe.MatchString(s) {
			return invalid("version must be major.minor.patch")
		}
	}
	if res, ok := m["result"]; ok {
		rm, ok := res.(map[string]any)
		if !ok {
			return invalid("result must be an object")
		}
		if err := validateResult(rm); err != nil {
			return err
		}
	}
	if ctx, ok := m["context"]; ok {
		cm, ok := ctx.(map[string]any)
		if !ok {
			return invalid("context must be an object")
		}
		if err := validateContext(cm, obj, version); err != nil {
			return err
		}
	}
	if atts, ok := m["attachments"]; ok {
		if err := validateAttachments(atts); err != nil {
			return err
		}
	}
	return nil
}

func validateVerb(v map[string]any) error {
	for k := range v {
		if k != "id" && k != "display" {
			return invalid("unknown verb property %q", k)
		}
	}
	id, ok := v["id"].(string)
	if !ok || !validIRI(id) {
		return invalid("verb.id must be an IRI")
	}
	if d, ok := v["display"]; ok {
		if err := validateLangMap(d, "verb.display"); err != nil {
			return err
		}
	}
	return nil
}

func validateObject(obj map[string]any, version string, now time.Time) error {
	ot, _ := obj["objectType"].(string)
	switch ot {
	case "", "Activity":
		return validateActivity(obj)
	case "Agent":
		return validateAgent(obj, "object", false)
	case "Group":
		return validateAgent(obj, "object", true)
	case "StatementRef":
		id, _ := obj["id"].(string)
		if !uuidRe.MatchString(id) {
			return invalid("StatementRef id must be a UUID")
		}
		for k := range obj {
			if k != "objectType" && k != "id" {
				return invalid("unknown StatementRef property %q", k)
			}
		}
		return nil
	case "SubStatement":
		return validateStatement(obj, version, now, true)
	default:
		return invalid("invalid objectType %q", ot)
	}
}

func validateActivity(obj map[string]any) error {
	for k := range obj {
		switch k {
		case "objectType", "id", "definition":
		default:
			return invalid("unknown activity property %q", k)
		}
	}
	id, ok := obj["id"].(string)
	if !ok || !validIRI(id) {
		return invalid("activity id must be an IRI")
	}
	if def, ok := obj["definition"]; ok {
		dm, ok := def.(map[string]any)
		if !ok {
			return invalid("activity definition must be an object")
		}
		if err := validateDefinition(dm); err != nil {
			return err
		}
	}
	return nil
}

func validateDefinition(d map[string]any) error {
	for k := range d {
		switch k {
		case "name", "description", "type", "moreInfo", "extensions", "interactionType",
			"correctResponsesPattern", "choices", "scale", "source", "target", "steps":
		default:
			return invalid("unknown definition property %q", k)
		}
	}
	if n, ok := d["name"]; ok {
		if err := validateLangMap(n, "definition.name"); err != nil {
			return err
		}
	}
	if n, ok := d["description"]; ok {
		if err := validateLangMap(n, "definition.description"); err != nil {
			return err
		}
	}
	if t, ok := d["type"]; ok {
		s, ok := t.(string)
		if !ok || !validIRI(s) {
			return invalid("definition.type must be an IRI")
		}
	}
	if t, ok := d["moreInfo"]; ok {
		s, ok := t.(string)
		if !ok || !validIRI(s) {
			return invalid("definition.moreInfo must be an IRI")
		}
	}
	if ext, ok := d["extensions"]; ok {
		if err := validateExtensions(ext); err != nil {
			return err
		}
	}
	if it, ok := d["interactionType"]; ok {
		s, ok := it.(string)
		if !ok || !interactionTypes[s] {
			return invalid("invalid interactionType")
		}
	}
	for _, k := range []string{"choices", "scale", "source", "target", "steps", "correctResponsesPattern"} {
		if v, ok := d[k]; ok {
			if _, ok := v.([]any); !ok {
				return invalid("%s must be an array", k)
			}
		}
	}
	return nil
}

func validateAgent(m map[string]any, where string, allowGroup bool) error {
	ot, _ := m["objectType"].(string)
	if ot == "" {
		ot = "Agent"
	}
	switch ot {
	case "Agent":
	case "Group":
		if !allowGroup {
			return invalid("%s must not be a Group", where)
		}
	default:
		return invalid("%s has invalid objectType %q", where, ot)
	}
	for k := range m {
		switch k {
		case "objectType", "name", "mbox", "mbox_sha1sum", "openid", "account", "member":
		default:
			return invalid("unknown %s property %q", where, k)
		}
	}
	if n, ok := m["name"]; ok {
		if _, ok := n.(string); !ok {
			return invalid("%s.name must be a string", where)
		}
	}
	ifis := 0
	if v, ok := m["mbox"]; ok {
		s, ok := v.(string)
		if !ok || !validMbox(s) {
			return invalid("%s.mbox must be a mailto IRI", where)
		}
		ifis++
	}
	if v, ok := m["mbox_sha1sum"]; ok {
		s, ok := v.(string)
		if !ok || !sha1Re.MatchString(s) {
			return invalid("%s.mbox_sha1sum must be 40 hex characters", where)
		}
		ifis++
	}
	if v, ok := m["openid"]; ok {
		s, ok := v.(string)
		if !ok || !validIRI(s) {
			return invalid("%s.openid must be an IRI", where)
		}
		ifis++
	}
	if v, ok := m["account"]; ok {
		acc, ok := v.(map[string]any)
		if !ok {
			return invalid("%s.account must be an object", where)
		}
		for k := range acc {
			if k != "homePage" && k != "name" {
				return invalid("unknown account property %q", k)
			}
		}
		home, _ := acc["homePage"].(string)
		name, _ := acc["name"].(string)
		if !validIRI(home) || strings.TrimSpace(name) == "" {
			return invalid("%s.account requires homePage IRI and name", where)
		}
		ifis++
	}
	members, hasMembers := m["member"]
	if ot == "Agent" {
		if hasMembers {
			return invalid("%s Agent must not have member", where)
		}
		if ifis != 1 {
			return invalid("%s must have exactly one inverse functional identifier", where)
		}
		return nil
	}
	if hasMembers {
		arr, ok := members.([]any)
		if !ok || len(arr) == 0 {
			return invalid("%s.member must be a non-empty array", where)
		}
		for _, item := range arr {
			am, ok := item.(map[string]any)
			if !ok {
				return invalid("%s.member entries must be agents", where)
			}
			if err := validateAgent(am, where+".member", false); err != nil {
				return err
			}
		}
	}
	if ifis == 0 {
		if !hasMembers {
			return invalid("%s Group must identify itself or list members", where)
		}
		return nil
	}
	if ifis != 1 {
		return invalid("%s must have exactly one inverse functional identifier", where)
	}
	return nil
}

func validateResult(m map[string]any) error {
	for k := range m {
		switch k {
		case "score", "success", "completion", "response", "duration", "extensions":
		default:
			return invalid("unknown result property %q", k)
		}
	}
	if s, ok := m["score"]; ok {
		sm, ok := s.(map[string]any)
		if !ok {
			return invalid("result.score must be an object")
		}
		if err := validateScore(sm); err != nil {
			return err
		}
	}
	for _, k := range []string{"success", "completion"} {
		if v, ok := m[k]; ok {
			if _, ok := v.(bool); !ok {
				return invalid("result.%s must be a boolean", k)
			}
		}
	}
	if v, ok := m["response"]; ok {
		if _, ok := v.(string); !ok {
			return invalid("result.response must be a string")
		}
	}
	if v, ok := m["duration"]; ok {
		s, ok := v.(string)
		if !ok || !validDuration(s) {
			return invalid("result.duration must be an ISO 8601 duration")
		}
	}
	if ext, ok := m["extensions"]; ok {
		return validateExtensions(ext)
	}
	return nil
}

func validateScore(m map[string]any) error {
	var raw, min, max *float64
	for k, v := range m {
		switch k {
		case "scaled", "raw", "min", "max":
			n, ok := asFloat(v)
			if !ok {
				return invalid("result.score.%s must be a number", k)
			}
			switch k {
			case "scaled":
				if n < -1 || n > 1 {
					return invalid("result.score.scaled must be between -1 and 1")
				}
			case "raw":
				raw = &n
			case "min":
				min = &n
			case "max":
				max = &n
			}
		default:
			return invalid("unknown score property %q", k)
		}
	}
	if min != nil && max != nil && *min > *max {
		return invalid("result.score.min must not exceed max")
	}
	if raw != nil && min != nil && *raw < *min {
		return invalid("result.score.raw is below min")
	}
	if raw != nil && max != nil && *raw > *max {
		return invalid("result.score.raw is above max")
	}
	return nil
}

func validateContext(m map[string]any, obj map[string]any, version string) error {
	for k := range m {
		switch k {
		case "registration", "instructor", "team", "contextActivities", "revision", "platform",
			"language", "statement", "extensions", "contextAgents", "contextGroups":
		default:
			return invalid("unknown context property %q", k)
		}
	}
	if Major(version) != "2" {
		if _, ok := m["contextAgents"]; ok {
			return invalid("contextAgents requires xAPI 2.0")
		}
		if _, ok := m["contextGroups"]; ok {
			return invalid("contextGroups requires xAPI 2.0")
		}
	}
	if v, ok := m["registration"]; ok {
		s, ok := v.(string)
		if !ok || !uuidRe.MatchString(s) {
			return invalid("context.registration must be a UUID")
		}
	}
	ot, _ := obj["objectType"].(string)
	if ot == "" {
		ot = "Activity"
	}
	if _, ok := m["revision"]; ok && ot != "Activity" {
		return invalid("context.revision is only valid when object is an Activity")
	}
	if _, ok := m["platform"]; ok && ot != "Activity" {
		return invalid("context.platform is only valid when object is an Activity")
	}
	for _, k := range []string{"revision", "platform", "language"} {
		if v, ok := m[k]; ok {
			if _, ok := v.(string); !ok {
				return invalid("context.%s must be a string", k)
			}
		}
	}
	if v, ok := m["instructor"]; ok {
		im, ok := v.(map[string]any)
		if !ok {
			return invalid("context.instructor must be an agent")
		}
		if err := validateAgent(im, "instructor", true); err != nil {
			return err
		}
	}
	if v, ok := m["team"]; ok {
		tm, ok := v.(map[string]any)
		if !ok {
			return invalid("context.team must be a group")
		}
		got, _ := tm["objectType"].(string)
		if got != "Group" {
			return invalid("context.team must be a Group")
		}
		if err := validateAgent(tm, "team", true); err != nil {
			return err
		}
	}
	if v, ok := m["statement"]; ok {
		sm, ok := v.(map[string]any)
		if !ok {
			return invalid("context.statement must be a StatementRef")
		}
		if err := validateObject(map[string]any{"objectType": "StatementRef", "id": sm["id"]}, version, time.Time{}); err != nil {
			sot, _ := sm["objectType"].(string)
			if sot != "StatementRef" {
				return invalid("context.statement must be a StatementRef")
			}
			return err
		}
		if sot, _ := sm["objectType"].(string); sot != "" && sot != "StatementRef" {
			return invalid("context.statement must be a StatementRef")
		}
	}
	if v, ok := m["contextActivities"]; ok {
		cm, ok := v.(map[string]any)
		if !ok {
			return invalid("contextActivities must be an object")
		}
		for k, item := range cm {
			switch k {
			case "parent", "grouping", "category", "other":
			default:
				return invalid("unknown contextActivities property %q", k)
			}
			arr, ok := item.([]any)
			if !ok {
				return invalid("contextActivities.%s must be an array", k)
			}
			for _, a := range arr {
				am, ok := a.(map[string]any)
				if !ok {
					return invalid("contextActivities entries must be activities")
				}
				if err := validateActivity(am); err != nil {
					return err
				}
			}
		}
	}
	if ext, ok := m["extensions"]; ok {
		if err := validateExtensions(ext); err != nil {
			return err
		}
	}
	if v, ok := m["contextAgents"]; ok {
		if err := validateContextActors(v, false); err != nil {
			return err
		}
	}
	if v, ok := m["contextGroups"]; ok {
		if err := validateContextActors(v, true); err != nil {
			return err
		}
	}
	return nil
}

func validateContextActors(v any, group bool) error {
	arr, ok := v.([]any)
	if !ok {
		if group {
			return invalid("contextGroups must be an array")
		}
		return invalid("contextAgents must be an array")
	}
	want := "contextAgent"
	field := "agent"
	label := "contextAgents"
	if group {
		want = "contextGroup"
		field = "group"
		label = "contextGroups"
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return invalid("%s entries must be objects", label)
		}
		ot, _ := m["objectType"].(string)
		if ot != want {
			return invalid("%s.objectType must be %s", label, want)
		}
		inner, ok := m[field].(map[string]any)
		if !ok {
			return invalid("%s.%s is required", label, field)
		}
		if err := validateAgent(inner, field, group); err != nil {
			return err
		}
		if group {
			if got, _ := inner["objectType"].(string); got != "Group" {
				return invalid("contextGroups.group must be a Group")
			}
		}
		if rt, ok := m["relevantTypes"]; ok {
			types, ok := rt.([]any)
			if !ok {
				return invalid("relevantTypes must be an array")
			}
			for _, t := range types {
				s, ok := t.(string)
				if !ok || !validIRI(s) {
					return invalid("relevantTypes entries must be IRIs")
				}
			}
		}
	}
	return nil
}

func validateAttachments(v any) error {
	arr, ok := v.([]any)
	if !ok {
		return invalid("attachments must be an array")
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return invalid("attachment must be an object")
		}
		for k := range m {
			switch k {
			case "usageType", "display", "description", "contentType", "length", "sha2", "fileUrl":
			default:
				return invalid("unknown attachment property %q", k)
			}
		}
		usage, _ := m["usageType"].(string)
		if !validIRI(usage) {
			return invalid("attachment.usageType must be an IRI")
		}
		if err := validateLangMap(m["display"], "attachment.display"); err != nil {
			return err
		}
		if d, ok := m["description"]; ok {
			if err := validateLangMap(d, "attachment.description"); err != nil {
				return err
			}
		}
		ct, _ := m["contentType"].(string)
		if strings.TrimSpace(ct) == "" {
			return invalid("attachment.contentType is required")
		}
		length, ok := asFloat(m["length"])
		if !ok || length < 0 || length != float64(int64(length)) {
			return invalid("attachment.length must be a non-negative integer")
		}
		sha, _ := m["sha2"].(string)
		if !sha2Re.MatchString(sha) {
			return invalid("attachment.sha2 must be a SHA-256 hex digest")
		}
		if fu, ok := m["fileUrl"]; ok {
			s, ok := fu.(string)
			if !ok || !validIRI(s) {
				return invalid("attachment.fileUrl must be an IRI")
			}
		}
	}
	return nil
}

func validateLangMap(v any, where string) error {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return invalid("%s must be a language map", where)
	}
	for _, val := range m {
		if _, ok := val.(string); !ok {
			return invalid("%s values must be strings", where)
		}
	}
	return nil
}

func validateExtensions(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return invalid("extensions must be an object")
	}
	for k := range m {
		if !validIRI(k) {
			return invalid("extension keys must be IRIs")
		}
	}
	return nil
}

func validIRI(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		return false
	}
	if u.Host == "" && u.Opaque == "" && u.Path == "" {
		return false
	}
	return true
}

func validMbox(s string) bool {
	if !strings.HasPrefix(strings.ToLower(s), "mailto:") {
		return false
	}
	rest := s[len("mailto:"):]
	return strings.Contains(rest, "@") && !strings.ContainsAny(rest, " \t")
}

func validDuration(s string) bool {
	if !durationRe.MatchString(s) {
		return false
	}
	return strings.ContainsAny(s, "0123456789")
}

func parseStamp(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("not a string")
	}
	return time.Parse(time.RFC3339Nano, s)
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case jsonNumber:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// jsonNumber is satisfied by encoding/json.Number without importing it in signatures.
type jsonNumber interface {
	Float64() (float64, error)
}
