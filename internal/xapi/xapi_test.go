package xapi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestNegotiate(t *testing.T) {
	got, ok := Negotiate("1.0.1")
	if !ok || got != Version103 {
		t.Fatalf("1.0.1 -> %s %v", got, ok)
	}
	got, ok = Negotiate("2.0")
	if !ok || got != Version200 {
		t.Fatalf("2.0 -> %s %v", got, ok)
	}
	if _, ok := Negotiate("1.1.0"); ok {
		t.Fatal("expected 1.1 to be rejected")
	}
	if _, ok := Negotiate(""); ok {
		t.Fatal("expected empty to be rejected")
	}
}

func TestPrepareRejectsMissingActor(t *testing.T) {
	_, err := Prepare(json.RawMessage(`{"verb":{"id":"http://adlnet.gov/expapi/verbs/voided"},"object":{"id":"http://example.com/a"}}`), Version103, time.Now(), "key", "http://lrs.example", "")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPrepareStampsLRSProperties(t *testing.T) {
	raw := json.RawMessage(`{"actor":{"mbox":"mailto:Alice@Example.org"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced"},"object":{"id":"http://example.com/activities/1"}}`)
	got, err := Prepare(raw, Version200, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), "client-key", "http://lrs.example", "")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(got.Statement.RawJSON, &m); err != nil {
		t.Fatal(err)
	}
	if m["version"] != Version200 {
		t.Fatalf("version %v", m["version"])
	}
	if m["stored"] == "" {
		t.Fatal("missing stored")
	}
	actor := m["actor"].(map[string]any)
	if actor["mbox"] != "mailto:alice@example.org" {
		t.Fatalf("mbox %v", actor["mbox"])
	}
	if got.Statement.ActorMBox == nil || *got.Statement.ActorMBox != "mailto:alice@example.org" {
		t.Fatal("actor mbox not extracted")
	}
	if len(got.Statement.Agents) != 1 {
		t.Fatalf("agents %+v", got.Statement.Agents)
	}
}

func TestContextAgentsRequire2(t *testing.T) {
	raw := json.RawMessage(`{
		"actor":{"mbox":"mailto:a@example.org"},
		"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced"},
		"object":{"id":"http://example.com/activities/1"},
		"context":{"contextAgents":[{"objectType":"contextAgent","agent":{"mbox":"mailto:b@example.org"},"relevantTypes":["http://example.com/rel"]}]}
	}`)
	if _, err := Prepare(raw, Version103, time.Now(), "k", "http://lrs.example", ""); err == nil {
		t.Fatal("expected 1.0.3 to reject contextAgents")
	}
	got, err := Prepare(raw, Version200, time.Now(), "k", "http://lrs.example", "")
	if err != nil {
		t.Fatalf("2.0 prepare: %v", err)
	}
	if len(got.Statement.Agents) < 2 {
		t.Fatalf("expected actor and context agent, got %+v", got.Statement.Agents)
	}
}

func TestVerifySignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "lrs-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	statement := map[string]any{
		"actor":  map[string]any{"mbox": "mailto:a@example.org"},
		"verb":   map[string]any{"id": "http://adlnet.gov/expapi/verbs/experienced"},
		"object": map[string]any{"id": "http://example.com/activities/1"},
	}
	payload, _ := json.Marshal(statement)
	header := map[string]any{"alg": "RS256", "x5c": []string{base64.StdEncoding.EncodeToString(cert.Raw)}}
	headerJSON, _ := json.Marshal(header)
	h64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	p64 := base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.New()
	_, _ = sum.Write([]byte(h64 + "." + p64))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}
	jws := h64 + "." + p64 + "." + base64.RawURLEncoding.EncodeToString(sig)
	signed := map[string]any{}
	for k, v := range statement {
		signed[k] = v
	}
	signed["id"] = "11111111-1111-1111-1111-111111111111"
	signed["stored"] = "2020-01-01T00:00:00.000Z"
	if err := VerifySignature(signed, jws, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	signed["actor"] = map[string]any{"mbox": "mailto:other@example.org"}
	if err := VerifySignature(signed, jws, true, time.Now()); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestAgentKeyStable(t *testing.T) {
	a := map[string]any{"mbox": "mailto:A@Example.org"}
	b := map[string]any{"mbox": "mailto:a@example.org"}
	ka, err := AgentKey(a)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := AgentKey(b)
	if err != nil {
		t.Fatal(err)
	}
	if ka != kb {
		t.Fatalf("keys differ %s %s", ka, kb)
	}
	if strings.TrimSpace(ka) == "" {
		t.Fatal("empty key")
	}
}
