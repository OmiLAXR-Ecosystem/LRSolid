package sso

import "strings"

// Endpoints are absolute URLs built from the public app origin. They must match
// the values registered at the IdP, so they come from configuration rather than
// the request Host header.
func OIDCRedirectURL(appURL, slug string) string {
	return strings.TrimRight(appURL, "/") + "/api/auth/oidc/" + slug + "/callback"
}

func SAMLACSURL(appURL, slug string) string {
	return strings.TrimRight(appURL, "/") + "/api/auth/saml/" + slug + "/acs"
}

func SAMLMetadataURL(appURL, slug string) string {
	return strings.TrimRight(appURL, "/") + "/api/auth/saml/" + slug + "/metadata"
}
