package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCSettings is the non-secret half of an OpenID Connect provider.
type OIDCSettings struct {
	Issuer               string            `json:"issuer"`
	ClientID             string            `json:"clientId"`
	Scopes               []string          `json:"scopes"`
	ClaimMappings        map[string]string `json:"claimMappings"`
	RoleMapping          map[string]string `json:"roleMapping"`
	RequireEmailVerified bool              `json:"requireEmailVerified"`
	TrustEmail           bool              `json:"trustEmail"`
}

func ParseOIDC(raw json.RawMessage) (OIDCSettings, error) {
	var s OIDCSettings
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &s); err != nil {
			return OIDCSettings{}, err
		}
	}
	if len(s.Scopes) == 0 {
		s.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	hasOpenID := false
	for _, scope := range s.Scopes {
		if scope == oidc.ScopeOpenID {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		s.Scopes = append([]string{oidc.ScopeOpenID}, s.Scopes...)
	}
	if s.ClaimMappings == nil {
		s.ClaimMappings = map[string]string{}
	}
	return s, nil
}

func (s OIDCSettings) claim(name, fallback string) string {
	if v := strings.TrimSpace(s.ClaimMappings[name]); v != "" {
		return v
	}
	return fallback
}

func httpClientContext(ctx context.Context) context.Context {
	return oidc.ClientContext(ctx, &http.Client{Timeout: 15 * time.Second})
}

func oidcProvider(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (*oidc.Provider, *oauth2.Config, *oidc.IDTokenVerifier, error) {
	ctx = httpClientContext(ctx)
	provider, err := oidc.NewProvider(ctx, strings.TrimRight(issuer, "/"))
	if err != nil {
		return nil, nil, nil, err
	}
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	return provider, conf, verifier, nil
}

// StartOIDC begins an Authorization Code + PKCE login.
func StartOIDC(ctx context.Context, p Provider, clientSecret, redirectURL, returnTo, invite string) (AuthRequest, string, error) {
	settings, err := ParseOIDC(p.Config)
	if err != nil {
		return AuthRequest{}, "", err
	}
	if settings.Issuer == "" || settings.ClientID == "" {
		return AuthRequest{}, "", ErrInvalidProvider
	}
	_, conf, _, err := oidcProvider(ctx, settings.Issuer, settings.ClientID, clientSecret, redirectURL, settings.Scopes)
	if err != nil {
		return AuthRequest{}, "", err
	}
	state, err := randomID()
	if err != nil {
		return AuthRequest{}, "", err
	}
	nonce, err := randomID()
	if err != nil {
		return AuthRequest{}, "", err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := conf.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oidc.Nonce(nonce),
	)
	req := AuthRequest{
		ID:           state,
		ProviderID:   p.ID,
		OrgID:        p.OrgID,
		Nonce:        nonce,
		PKCEVerifier: verifier,
		ReturnTo:     SafeReturnTo(returnTo),
		InviteToken:  invite,
		ExpiresAt:    time.Now().Add(requestTTL),
	}
	return req, authURL, nil
}

// FinishOIDC exchanges the code and verifies the ID token.
func FinishOIDC(ctx context.Context, p Provider, clientSecret, redirectURL string, req AuthRequest, code string) (Result, error) {
	settings, err := ParseOIDC(p.Config)
	if err != nil {
		return Result{}, err
	}
	provider, conf, verifier, err := oidcProvider(ctx, settings.Issuer, settings.ClientID, clientSecret, redirectURL, settings.Scopes)
	if err != nil {
		return Result{}, err
	}
	ctx = httpClientContext(ctx)
	tok, err := conf.Exchange(ctx, code, oauth2.VerifierOption(req.PKCEVerifier))
	if err != nil {
		return Result{}, err
	}
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		return Result{}, ErrInvalidProvider
	}
	idToken, err := verifier.Verify(ctx, rawID)
	if err != nil {
		return Result{}, err
	}
	if idToken.Nonce != req.Nonce {
		return Result{}, ErrReplay
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return Result{}, err
	}
	if stringClaim(claims, settings.claim("email", "email")) == "" {
		if info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(tok)); err == nil && info != nil {
			extra := map[string]any{}
			if err := info.Claims(&extra); err == nil {
				for k, v := range extra {
					if _, exists := claims[k]; !exists {
						claims[k] = v
					}
				}
			}
		}
	}
	identity := identityFromOIDC(settings, idToken.Subject, claims)
	if settings.RequireEmailVerified && !identity.EmailVerified {
		return Result{}, ErrEmailUnverified
	}
	var sessionFor time.Duration
	if !idToken.Expiry.IsZero() {
		sessionFor = time.Until(idToken.Expiry)
	}
	return Result{Identity: identity, SessionFor: sessionFor}, nil
}

func identityFromOIDC(settings OIDCSettings, subject string, claims map[string]any) Identity {
	if mapped := stringClaim(claims, settings.claim("subject", "")); mapped != "" {
		subject = mapped
	}
	email := stringClaim(claims, settings.claim("email", "email"))
	verified, ok := boolClaim(claims, "email_verified")
	if settings.TrustEmail && email != "" {
		verified = true
	} else if !ok {
		verified = false
	}
	name := stringClaim(claims, settings.claim("displayName", "name"))
	groupClaim := settings.claim("groups", "groups")
	groups := stringListClaim(claims, groupClaim)
	if len(groups) == 0 && groupClaim == "groups" {
		groups = stringListClaim(claims, "roles")
	}
	return Identity{
		Subject:       subject,
		Email:         email,
		EmailVerified: verified,
		DisplayName:   name,
		Groups:        groups,
		Raw:           claims,
	}
}

// TestOIDC checks that the issuer publishes discovery metadata and JWKS.
func TestOIDC(ctx context.Context, raw json.RawMessage) (string, error) {
	settings, err := ParseOIDC(raw)
	if err != nil {
		return "", err
	}
	if settings.Issuer == "" || settings.ClientID == "" {
		return "", ErrInvalidProvider
	}
	provider, _, _, err := oidcProvider(ctx, settings.Issuer, settings.ClientID, "", "http://localhost/callback", settings.Scopes)
	if err != nil {
		return "", err
	}
	return "issuer " + provider.Endpoint().AuthURL, nil
}
