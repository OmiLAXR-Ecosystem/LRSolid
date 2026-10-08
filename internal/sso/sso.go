// Package sso implements human login via OpenID Connect and SAML 2.0
// (Shibboleth). Successful logins resolve to an LRSolid user; the HTTP layer
// then issues the existing session cookie.
package sso

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	TypeOIDC = "oidc"
	TypeSAML = "saml"

	requestTTL = 10 * time.Minute
)

var (
	ErrNotFound         = errors.New("not found")
	ErrExpired          = errors.New("expired")
	ErrNoAccount        = errors.New("no account")
	ErrDisabled         = errors.New("account disabled")
	ErrEmailUnverified  = errors.New("email unverified")
	ErrEmailRequired    = errors.New("email required")
	ErrTransientSubject = errors.New("transient subject")
	ErrReplay           = errors.New("assertion replay")
	ErrProviderDisabled = errors.New("provider disabled")
	ErrInvalidProvider  = errors.New("invalid provider")
)

// Provider is one configured OIDC or SAML login method.
type Provider struct {
	ID            string
	OrgID         *string
	OrgSlug       string
	Slug          string
	Type          string
	DisplayName   string
	Enabled       bool
	Config        json.RawMessage
	SecretRef     *string
	AutoProvision bool
	DefaultRole   string
	LinkByEmail   bool
}

// Identity is the normalized result of an OIDC or SAML login.
type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	Groups        []string
	Raw           map[string]any
}

// AuthRequest is an in-flight login, stored server-side so the SAML ACS
// cross-site POST does not depend on a SameSite cookie.
type AuthRequest struct {
	ID           string
	ProviderID   string
	OrgID        *string
	Nonce        string
	PKCEVerifier string
	ReturnTo     string
	InviteToken  string
	ExpiresAt    time.Time
}

// LoginProvider is the public description shown on the sign-in page.
type LoginProvider struct {
	Slug        string `json:"slug"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
}

// Result is a completed provider login, before user resolution.
type Result struct {
	Identity   Identity
	SessionFor time.Duration
}

// SafeReturnTo keeps post-login redirects on this site.
func SafeReturnTo(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return "/"
	}
	if strings.Contains(raw, "://") || strings.ContainsAny(raw, "\r\n") {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" {
		return "/"
	}
	if u.Path == "" {
		return "/"
	}
	return u.RequestURI()
}

func roleRank(role string) int {
	switch role {
	case "viewer":
		return 1
	case "member":
		return 2
	case "admin":
		return 3
	case "owner":
		return 4
	default:
		return 0
	}
}

// MapRole picks an org role from IdP groups. sync is true when a mapping
// matched and the membership should be updated on every login.
func MapRole(groups []string, mapping map[string]string, fallback string) (role string, sync bool) {
	if roleRank(fallback) == 0 {
		fallback = "viewer"
	}
	if len(mapping) == 0 {
		return fallback, false
	}
	best := ""
	bestRank := 0
	for _, g := range groups {
		mapped, ok := mapping[g]
		if !ok {
			continue
		}
		if rank := roleRank(mapped); rank > bestRank {
			best = mapped
			bestRank = rank
		}
	}
	if best == "" {
		if mapped, ok := mapping["*"]; ok && roleRank(mapped) > 0 {
			return mapped, true
		}
		return fallback, false
	}
	return best, true
}

func stringClaim(claims map[string]any, key string) string {
	if key == "" {
		return ""
	}
	v, ok := claims[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func boolClaim(claims map[string]any, key string) (bool, bool) {
	v, ok := claims[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func stringListClaim(claims map[string]any, key string) []string {
	v, ok := claims[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{strings.TrimSpace(t)}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func appendUnique(dst []string, vals ...string) []string {
	seen := map[string]struct{}{}
	for _, v := range dst {
		seen[v] = struct{}{}
	}
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		dst = append(dst, v)
	}
	return dst
}
