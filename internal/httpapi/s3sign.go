package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"
)

// signS3Request signs an HTTP request with AWS Signature Version 4 (service "s3").
// payloadSHA256 is the lowercase hex SHA-256 of the request body; pass
// emptyPayloadSHA256 for bodyless requests (GET/HEAD/DELETE).
//
// Only what path-style S3 requests need is implemented: no query-string
// canonicalisation beyond sorting, no session tokens, no chunked signing.
func signS3Request(req *http.Request, region, accessKey, secretKey, payloadSHA256 string, now time.Time) {
	amzDate := now.UTC().Format("20060102T150405Z")
	dateStamp := now.UTC().Format("20060102")

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadSHA256)

	// Canonical headers: host + all x-amz-* headers, lowercase, sorted.
	canonical := map[string]string{"host": req.URL.Host}
	for k, v := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz-") {
			canonical[lk] = strings.TrimSpace(strings.Join(v, ","))
		}
	}
	names := make([]string, 0, len(canonical))
	for k := range canonical {
		names = append(names, k)
	}
	sort.Strings(names)

	var canonicalHeaders strings.Builder
	for _, k := range names {
		canonicalHeaders.WriteString(k + ":" + canonical[k] + "\n")
	}
	signedHeaders := strings.Join(names, ";")

	canonicalQuery := canonicalQueryString(req.URL.Query())
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadSHA256,
	}, "\n")

	scope := dateStamp + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+
			", SignedHeaders="+signedHeaders+
			", Signature="+signature)
}

const emptyPayloadSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func canonicalQueryString(values map[string][]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := append([]string(nil), values[k]...)
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, awsURIEncode(k)+"="+awsURIEncode(v))
		}
	}
	return strings.Join(parts, "&")
}

// awsURIEncode implements the AWS SigV4 URI encoding rules: unreserved
// characters stay literal, everything else becomes %XX (space is %20, not +).
func awsURIEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteString("%" + strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}

func hexSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
