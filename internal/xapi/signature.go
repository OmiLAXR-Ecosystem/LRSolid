package xapi

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash"
	"reflect"
	"strings"
	"time"
)

// VerifySignature checks a JWS compact serialization against the statement.
// requireX5C is set for xAPI 2.0 requests. The comparison ignores id, stored,
// authority, version, and the signature attachment itself.
func VerifySignature(statement map[string]any, jws string, requireX5C bool, now time.Time) error {
	parts := strings.Split(jws, ".")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return invalid("signature attachment is not a compact JWS")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return invalid("signature header is not base64url")
	}
	var header map[string]any
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return invalid("signature header is not JSON")
	}
	alg, _ := header["alg"].(string)
	hashFn, err := hashForAlg(alg)
	if err != nil {
		return err
	}
	pub, err := publicKeyFromHeader(header, requireX5C, now)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return invalid("signature is not base64url")
	}
	_, _ = hashFn.Write([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, cryptoHash(alg), hashFn.Sum(nil), sig); err != nil {
		return invalid("signature verification failed")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return invalid("signature payload is not base64url")
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return invalid("signature payload is not a statement")
	}
	if !sameSignedStatement(statement, payload) {
		return invalid("signature payload does not match the statement")
	}
	return nil
}

func hashForAlg(alg string) (hash.Hash, error) {
	switch alg {
	case "RS256":
		return sha256.New(), nil
	case "RS384":
		return sha512.New384(), nil
	case "RS512":
		return sha512.New(), nil
	default:
		return nil, invalid("signature algorithm must be RS256, RS384, or RS512")
	}
}

func cryptoHash(alg string) crypto.Hash {
	switch alg {
	case "RS384":
		return crypto.SHA384
	case "RS512":
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}

func publicKeyFromHeader(header map[string]any, requireX5C bool, now time.Time) (*rsa.PublicKey, error) {
	raw, has := header["x5c"]
	if !has {
		if requireX5C {
			return nil, invalid("signature x5c certificate chain is required")
		}
		return nil, invalid("signature is missing x5c")
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil, invalid("signature x5c must be a certificate chain")
	}
	certs := make([]*x509.Certificate, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, invalid("signature x5c entries must be strings")
		}
		der, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, invalid("signature certificate is not base64")
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, invalid("signature certificate is invalid")
		}
		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			return nil, invalid("signature certificate is not valid at this time")
		}
		certs = append(certs, cert)
	}
	for i := 0; i < len(certs)-1; i++ {
		if err := certs[i].CheckSignatureFrom(certs[i+1]); err != nil {
			return nil, invalid("signature certificate chain is invalid")
		}
	}
	pub, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, invalid("signature certificate must use an RSA key")
	}
	return pub, nil
}

func sameSignedStatement(statement, payload map[string]any) bool {
	a := unsignedView(statement)
	b := unsignedView(payload)
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	var ja, jb any
	_ = json.Unmarshal(ab, &ja)
	_ = json.Unmarshal(bb, &jb)
	return jsonEqual(ja, jb)
}

func unsignedView(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		switch k {
		case "id", "stored", "authority", "version":
			continue
		case "attachments":
			arr, _ := v.([]any)
			kept := make([]any, 0, len(arr))
			for _, item := range arr {
				am, ok := item.(map[string]any)
				if !ok {
					continue
				}
				usage, _ := am["usageType"].(string)
				if usage == AttachmentSignature {
					continue
				}
				kept = append(kept, am)
			}
			if len(kept) > 0 {
				out[k] = kept
			}
		default:
			out[k] = v
		}
	}
	return out
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	var ja, jb any
	_ = json.Unmarshal(ab, &ja)
	_ = json.Unmarshal(bb, &jb)
	return reflect.DeepEqual(ja, jb)
}

// CertDERBase64 returns the base64 DER of a PEM or DER certificate.
func CertDERBase64(cert *x509.Certificate) string {
	return base64.StdEncoding.EncodeToString(cert.Raw)
}

// ParseCertPEM decodes a PEM certificate.
func ParseCertPEM(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no pem block")
	}
	return x509.ParseCertificate(block.Bytes)
}
