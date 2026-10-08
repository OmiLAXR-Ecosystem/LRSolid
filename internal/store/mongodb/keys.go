package mongodb

import "strings"

// MongoDB refuses document keys that contain "." or start with "$".
// xAPI extension IRIs contain dots, so keys are escaped before they are stored
// in the queryable raw document. The original JSON is kept separately.
const (
	escapedDot    = "\uFF0E" // FULLWIDTH FULL STOP
	escapedDollar = "\uFF04" // FULLWIDTH DOLLAR SIGN
)

func escapeKey(k string) string {
	k = strings.ReplaceAll(k, ".", escapedDot)
	if strings.HasPrefix(k, "$") {
		k = escapedDollar + k[1:]
	}
	return k
}

func unescapeKey(k string) string {
	if strings.HasPrefix(k, escapedDollar) {
		k = "$" + k[len(escapedDollar):]
	}
	return strings.ReplaceAll(k, escapedDot, ".")
}

func escapeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[escapeKey(k)] = escapeValue(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = escapeValue(val)
		}
		return out
	default:
		return v
	}
}

func unescapeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[unescapeKey(k)] = unescapeValue(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = unescapeValue(val)
		}
		return out
	default:
		return v
	}
}
