package mongodb

import (
	"reflect"
	"testing"
)

func TestEscapeKeysRoundTrip(t *testing.T) {
	in := map[string]any{
		"context": map[string]any{
			"extensions": map[string]any{
				"http://example.com/ext": "x",
				"$score":                 1.0,
			},
		},
		"members": []any{
			map[string]any{"mbox": "mailto:a@x"},
		},
	}
	back := unescapeValue(escapeValue(in))
	if !reflect.DeepEqual(in, back) {
		t.Fatalf("round trip mismatch: %#v", back)
	}
	escaped := escapeValue(in).(map[string]any)
	ext := escaped["context"].(map[string]any)["extensions"].(map[string]any)
	if _, ok := ext["http://example.com/ext"]; ok {
		t.Fatal("dotted key was stored unchanged")
	}
}
