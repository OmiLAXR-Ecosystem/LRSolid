// Package auth provides JWT generation and validation using only the Go standard library.
// Tokens are signed with HMAC-SHA256 and stored in an httpOnly cookie.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenExpired = errors.New("token expired")
)

// Claims is the JWT payload stored inside the session cookie.
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
	Iat    int64  `json:"iat"`
}

var jwtHeader = b64url([]byte(`{"alg":"HS256","typ":"JWT"}`))

// GenerateToken creates a signed JWT for the given claims with the specified TTL.
func GenerateToken(secret string, c Claims, ttl time.Duration) (string, error) {
	now := time.Now().Unix()
	c.Iat = now
	c.Exp = now + int64(ttl.Seconds())

	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	unsigned := jwtHeader + "." + b64url(payload)
	return unsigned + "." + sign(secret, unsigned), nil
}

// ValidateToken parses and validates a JWT string, returning Claims on success.
func ValidateToken(secret, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrTokenInvalid
	}

	unsigned := parts[0] + "." + parts[1]
	expected := sign(secret, unsigned)
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return Claims{}, ErrTokenInvalid
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}

	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return Claims{}, ErrTokenInvalid
	}

	if time.Now().Unix() > c.Exp {
		return Claims{}, ErrTokenExpired
	}

	return c, nil
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return b64url(mac.Sum(nil))
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
