package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseStatementsPayload(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantLen     int
		wantErr     bool
	}{
		{
			name:        "wrapper statements array",
			contentType: "application/json",
			body:        `{"statements":[{"id":"a"},{"id":"b"}]}`,
			wantLen:     2,
		},
		{
			name:        "wrapper single statement",
			contentType: "application/json",
			body:        `{"statement":{"id":"a"}}`,
			wantLen:     1,
		},
		{
			name:        "raw single statement object",
			contentType: "application/json",
			body:        `{"id":"a","actor":{"mbox":"mailto:test@example.org"}}`,
			wantLen:     1,
		},
		{
			name:        "raw statement array",
			contentType: "application/json",
			body:        `[{"id":"a"},{"id":"b"}]`,
			wantLen:     2,
		},
		{
			name:        "form encoded statement",
			contentType: "application/x-www-form-urlencoded",
			body:        `statement=%7B%22id%22%3A%22a%22%7D`,
			wantLen:     1,
		},
		{
			name:        "form encoded statement with wrong content type",
			contentType: "text/plain",
			body:        `statement=%7B%22id%22%3A%22a%22%7D`,
			wantLen:     1,
		},
		{
			name:        "invalid body",
			contentType: "application/json",
			body:        `not-json-and-no-form`,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/xapi/statements", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			got, err := parseStatementsPayload(req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Fatalf("expected %d statements, got %d", tc.wantLen, len(got))
			}
		})
	}
}
