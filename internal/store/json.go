package store

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// JSONEqual reports whether two JSON documents have the same structure.
// Object key order is ignored. Invalid JSON falls back to byte equality.
func JSONEqual(a, b []byte) bool {
	var ja, jb any
	if err := json.Unmarshal(a, &ja); err != nil {
		return bytes.Equal(a, b)
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(ja, jb)
}

// StatementsEquivalent reports whether two stored statements are the same xAPI
// document. LRS-assigned properties (stored, authority, timestamp) are ignored,
// because a replay is not a conflict when only those differ.
func StatementsEquivalent(a, b []byte) bool {
	var ja, jb map[string]any
	if err := json.Unmarshal(a, &ja); err != nil {
		return JSONEqual(a, b)
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		return JSONEqual(a, b)
	}
	for _, k := range []string{"stored", "authority", "timestamp"} {
		delete(ja, k)
		delete(jb, k)
	}
	return reflect.DeepEqual(ja, jb)
}
