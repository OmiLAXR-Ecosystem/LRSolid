package store

import "testing"

func TestJSONEqualIgnoresKeyOrder(t *testing.T) {
	a := []byte(`{"id":"1","actor":{"mbox":"mailto:a@x","name":"A"}}`)
	b := []byte(`{"actor":{"name":"A","mbox":"mailto:a@x"},"id":"1"}`)
	if !JSONEqual(a, b) {
		t.Fatal("expected equal JSON")
	}
	c := []byte(`{"id":"1","actor":{"mbox":"mailto:b@x"}}`)
	if JSONEqual(a, c) {
		t.Fatal("expected different JSON")
	}
}
