package sso

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/saml"
)

func TestSafeReturnTo(t *testing.T) {
	cases := map[string]string{
		"":                        "/",
		"/default/statements":     "/default/statements",
		"/default/statements?q=1": "/default/statements?q=1",
		"//evil.example":          "/",
		"https://evil.example":    "/",
		"/\\evil.example":         "/",
		"statements":              "/",
	}
	for in, want := range cases {
		if got := SafeReturnTo(in); got != want {
			t.Errorf("SafeReturnTo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapRole(t *testing.T) {
	role, sync := MapRole([]string{"students"}, nil, "member")
	if role != "member" || sync {
		t.Fatalf("fallback: %s sync=%v", role, sync)
	}
	role, sync = MapRole([]string{"lrs-admins", "students"}, map[string]string{
		"students":   "viewer",
		"lrs-admins": "admin",
		"*":          "member",
	}, "viewer")
	if role != "admin" || !sync {
		t.Fatalf("highest match: %s sync=%v", role, sync)
	}
	role, sync = MapRole([]string{"other"}, map[string]string{"*": "member"}, "viewer")
	if role != "member" || !sync {
		t.Fatalf("wildcard: %s sync=%v", role, sync)
	}
	role, sync = MapRole(nil, map[string]string{"admins": "admin"}, "viewer")
	if role != "viewer" || sync {
		t.Fatalf("no match: %s sync=%v", role, sync)
	}
}

func TestIdentityFromAssertionEduPerson(t *testing.T) {
	assertion := &saml.Assertion{
		Subject: &saml.Subject{NameID: &saml.NameID{
			Format: string(saml.TransientNameIDFormat),
			Value:  "transient-1",
		}},
		AttributeStatements: []saml.AttributeStatement{{
			Attributes: []saml.Attribute{
				{Name: attrEPPN, Values: []saml.AttributeValue{{Value: "ada@uni.example"}}},
				{Name: attrMail, Values: []saml.AttributeValue{{Value: "ada@uni.example"}}},
				{Name: attrGivenName, Values: []saml.AttributeValue{{Value: "Ada"}}},
				{Name: attrSurname, Values: []saml.AttributeValue{{Value: "Lovelace"}}},
				{Name: attrEntitlement, Values: []saml.AttributeValue{{Value: "lrs-admins"}}},
			},
		}},
	}
	settings, err := ParseSAML(nil)
	if err != nil {
		t.Fatal(err)
	}
	id, err := IdentityFromAssertion(assertion, settings)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "ada@uni.example" || id.Email != "ada@uni.example" || id.DisplayName != "Ada Lovelace" {
		t.Fatalf("identity = %+v", id)
	}
	if !id.EmailVerified {
		t.Fatal("expected trusted saml email")
	}
	role, sync := MapRole(id.Groups, map[string]string{"lrs-admins": "admin"}, "viewer")
	if role != "admin" || !sync {
		t.Fatalf("role %s sync %v groups %v", role, sync, id.Groups)
	}
}

func TestIdentityFromAssertionRejectsTransientOnly(t *testing.T) {
	assertion := &saml.Assertion{
		Subject: &saml.Subject{NameID: &saml.NameID{
			Format: string(saml.TransientNameIDFormat),
			Value:  "transient-1",
		}},
	}
	_, err := IdentityFromAssertion(assertion, SAMLSettings{})
	if err != ErrTransientSubject {
		t.Fatalf("got %v", err)
	}
}

func TestMetadataXMLRequestsEduPerson(t *testing.T) {
	cert, key, err := GenerateSPCredentials("https://lrs.example/api/auth/saml/shibboleth/metadata")
	if err != nil {
		t.Fatal(err)
	}
	settings := SAMLSettings{
		SPEntityID: "https://lrs.example/api/auth/saml/shibboleth/metadata",
		SPCertPEM:  cert,
	}
	body, err := MetadataXML(settings, key, "https://lrs.example/api/auth/saml/shibboleth/acs", settings.SPEntityID)
	if err != nil {
		t.Fatal(err)
	}
	xml := string(body)
	for _, want := range []string{settings.SPEntityID, attrMail, attrEPPN, "AssertionConsumerService", "HTTP-POST"} {
		if !strings.Contains(xml, want) {
			t.Errorf("metadata missing %s", want)
		}
	}
	if strings.Contains(xml, "HTTP-Artifact") {
		t.Fatal("artifact binding should not be advertised")
	}
}

func TestOIDCCodeFlow(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var nonce string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/auth",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/keys",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{rsaJWK(key, "test")}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := signRS256(key, map[string]any{
			"iss":            issuer,
			"sub":            "user-1",
			"aud":            "lrsolid",
			"exp":            time.Now().Add(time.Hour).Unix(),
			"iat":            time.Now().Unix(),
			"nonce":          nonce,
			"email":          "ada@uni.example",
			"email_verified": true,
			"name":           "Ada",
			"groups":         []string{"lrs-admins"},
		})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "atk",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     token,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	raw, _ := json.Marshal(OIDCSettings{Issuer: issuer, ClientID: "lrsolid", Scopes: []string{"openid", "email"}})
	p := Provider{ID: "00000000-0000-0000-0000-000000000001", Slug: "oidc", Type: TypeOIDC, Config: raw}
	req, authURL, err := StartOIDC(context.Background(), p, "secret", "http://localhost/callback", "/default/statements", "")
	if err != nil {
		t.Fatal(err)
	}
	if authURL == "" || req.Nonce == "" || req.PKCEVerifier == "" || req.ReturnTo != "/default/statements" {
		t.Fatalf("start = %#v url %s", req, authURL)
	}
	nonce = req.Nonce
	result, err := FinishOIDC(context.Background(), p, "secret", "http://localhost/callback", req, "code")
	if err != nil {
		t.Fatal(err)
	}
	if result.Identity.Subject != "user-1" || result.Identity.Email != "ada@uni.example" || !result.Identity.EmailVerified {
		t.Fatalf("identity = %+v", result.Identity)
	}
	if len(result.Identity.Groups) != 1 || result.Identity.Groups[0] != "lrs-admins" {
		t.Fatalf("groups = %v", result.Identity.Groups)
	}
}

func signRS256(key *rsa.PrivateKey, claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT","kid":"test"}`))
	payload, _ := json.Marshal(claims)
	body := base64.RawURLEncoding.EncodeToString(payload)
	unsigned := header + "." + body
	sum := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func rsaJWK(key *rsa.PrivateKey, kid string) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}
