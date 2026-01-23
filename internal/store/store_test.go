package store

import (
	"testing"
	"time"
)

func TestCursorRoundtrip(t *testing.T) {
	c := Cursor{StoredAt: time.Unix(0, 123456789).UTC(), ID: StatementID("abc")}
	token := EncodeCursor(c)

	got, err := DecodeCursor(token)
	if err != nil {
		t.Fatalf("DecodeCursor error: %v", err)
	}
	if got.StoredAt.UnixNano() != c.StoredAt.UnixNano() || got.ID != c.ID {
		t.Fatalf("cursor mismatch: got=%v want=%v", got, c)
	}
}

func TestCursorInvalid(t *testing.T) {
	_, err := DecodeCursor("not-base64!!!")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestHashDeterministic(t *testing.T) {
	a := HashMBox("mailto:test@example.org")
	b := HashMBox("mailto:test@example.org")
	if a != b {
		t.Fatalf("expected deterministic hash")
	}
}
