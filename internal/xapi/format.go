package xapi

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// ApplyFormat rewrites a stored statement for the xAPI format parameter.
// lookup returns the canonical activity definition when the LRS has one.
func ApplyFormat(raw []byte, format string, acceptLanguage string, lookup func(activityID string) (json.RawMessage, bool)) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw, err
	}
	switch format {
	case "", "exact":
		return marshal(m)
	case "ids":
		stripIDs(m)
		return marshal(m)
	case "canonical":
		langs := parseAcceptLanguage(acceptLanguage)
		applyCanonical(m, langs, lookup)
		return marshal(m)
	default:
		return nil, invalid("format must be ids, exact, or canonical")
	}
}

func stripIDs(m map[string]any) {
	if verb, ok := m["verb"].(map[string]any); ok {
		delete(verb, "display")
	}
	if actor, ok := m["actor"].(map[string]any); ok {
		stripAgent(actor)
	}
	if obj, ok := m["object"].(map[string]any); ok {
		stripObject(obj)
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		stripContext(ctx)
	}
}

func stripAgent(m map[string]any) {
	delete(m, "name")
	if members, ok := m["member"].([]any); ok {
		for _, member := range members {
			if mm, ok := member.(map[string]any); ok {
				stripAgent(mm)
			}
		}
	}
}

func stripObject(obj map[string]any) {
	ot, _ := obj["objectType"].(string)
	switch ot {
	case "", "Activity":
		delete(obj, "definition")
	case "Agent", "Group":
		stripAgent(obj)
	case "SubStatement":
		stripIDs(obj)
	}
}

func stripContext(ctx map[string]any) {
	if inst, ok := ctx["instructor"].(map[string]any); ok {
		stripAgent(inst)
	}
	if team, ok := ctx["team"].(map[string]any); ok {
		stripAgent(team)
	}
	if ca, ok := ctx["contextActivities"].(map[string]any); ok {
		for _, key := range []string{"parent", "grouping", "category", "other"} {
			arr, _ := ca[key].([]any)
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					delete(m, "definition")
				}
			}
		}
	}
	if arr, ok := ctx["contextAgents"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if agent, ok := m["agent"].(map[string]any); ok {
					stripAgent(agent)
				}
			}
		}
	}
	if arr, ok := ctx["contextGroups"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if group, ok := m["group"].(map[string]any); ok {
					stripAgent(group)
				}
			}
		}
	}
}

func applyCanonical(m map[string]any, langs []langPref, lookup func(string) (json.RawMessage, bool)) {
	filterLangs(m, langs)
	if obj, ok := m["object"].(map[string]any); ok {
		canonicalObject(obj, langs, lookup)
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		if ca, ok := ctx["contextActivities"].(map[string]any); ok {
			for _, key := range []string{"parent", "grouping", "category", "other"} {
				arr, _ := ca[key].([]any)
				for _, item := range arr {
					if am, ok := item.(map[string]any); ok {
						replaceDefinition(am, langs, lookup)
					}
				}
			}
		}
	}
}

func canonicalObject(obj map[string]any, langs []langPref, lookup func(string) (json.RawMessage, bool)) {
	ot, _ := obj["objectType"].(string)
	switch ot {
	case "", "Activity":
		replaceDefinition(obj, langs, lookup)
	case "SubStatement":
		applyCanonical(obj, langs, lookup)
	}
}

func replaceDefinition(activity map[string]any, langs []langPref, lookup func(string) (json.RawMessage, bool)) {
	id, _ := activity["id"].(string)
	if id != "" && lookup != nil {
		if raw, ok := lookup(id); ok && len(raw) > 0 {
			var def map[string]any
			if json.Unmarshal(raw, &def) == nil {
				activity["definition"] = def
			}
		}
	}
	if def, ok := activity["definition"].(map[string]any); ok {
		filterLangs(def, langs)
	}
}

type langPref struct {
	tag string
	q   float64
}

func parseAcceptLanguage(header string) []langPref {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	var out []langPref
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			for _, param := range strings.Split(part[i+1:], ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					if n, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64); err == nil {
						q = n
					}
				}
			}
		}
		if tag == "" || tag == "*" {
			continue
		}
		out = append(out, langPref{tag: strings.ToLower(tag), q: q})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].q > out[j].q })
	return out
}

func filterLangs(v any, langs []langPref) {
	switch n := v.(type) {
	case map[string]any:
		for k, child := range n {
			if isLangMapKey(k) {
				if lm, ok := child.(map[string]any); ok {
					n[k] = pickLanguage(lm, langs)
				}
				continue
			}
			filterLangs(child, langs)
		}
	case []any:
		for _, child := range n {
			filterLangs(child, langs)
		}
	}
}

func isLangMapKey(k string) bool {
	switch k {
	case "display", "name", "description":
		return true
	default:
		return false
	}
}

func pickLanguage(m map[string]any, langs []langPref) map[string]any {
	if len(m) == 0 || len(langs) == 0 {
		return m
	}
	type cand struct {
		key  string
		rank int
		q    float64
	}
	var best *cand
	for key := range m {
		lk := strings.ToLower(key)
		for i, pref := range langs {
			rank := -1
			if lk == pref.tag {
				rank = 0
			} else if strings.HasPrefix(lk, pref.tag+"-") || strings.HasPrefix(pref.tag, lk+"-") {
				rank = 1
			}
			if rank < 0 {
				continue
			}
			c := cand{key: key, rank: rank, q: pref.q}
			if best == nil || c.q > best.q || (c.q == best.q && (c.rank < best.rank || (c.rank == best.rank && i < 0))) {
				best = &c
			}
			break
		}
	}
	if best == nil {
		return m
	}
	return map[string]any{best.key: m[best.key]}
}
