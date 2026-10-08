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
