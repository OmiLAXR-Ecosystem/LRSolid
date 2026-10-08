package sso

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	attrSubjectID         = "urn:oasis:names:tc:SAML:attribute:subject-id"
	attrPairwiseID        = "urn:oasis:names:tc:SAML:attribute:pairwise-id"
	attrEPPN              = "urn:oid:1.3.6.1.4.1.5923.1.1.1.6"
	attrMail              = "urn:oid:0.9.2342.19200300.100.1.3"
	attrDisplayName       = "urn:oid:2.16.840.1.113730.3.1.241"
	attrGivenName         = "urn:oid:2.5.4.42"
	attrSurname           = "urn:oid:2.5.4.4"
	attrAffiliation       = "urn:oid:1.3.6.1.4.1.5923.1.1.1.1"
	attrScopedAffiliation = "urn:oid:1.3.6.1.4.1.5923.1.1.1.9"
	attrEntitlement       = "urn:oid:1.3.6.1.4.1.5923.1.1.1.7"
	attrIsMemberOf        = "urn:oid:1.3.6.1.4.1.5923.1.5.1.1"
)

// SAMLSettings is the non-secret half of a SAML service provider.
// Shibboleth is this type with eduPerson attribute defaults.
type SAMLSettings struct {
	IDPMetadataURL    string            `json:"idpMetadataUrl"`
	IDPMetadataXML    string            `json:"idpMetadataXml"`
	SPEntityID        string            `json:"spEntityId"`
	SPCertPEM         string            `json:"spCertPem"`
	NameIDFormat      string            `json:"nameIdFormat"`
	AttributeMapping  map[string]string `json:"attributeMappings"`
	RoleMapping       map[string]string `json:"roleMapping"`
	SignRequests      *bool             `json:"signRequests"`
	AllowIDPInitiated bool              `json:"allowIdpInitiated"`
	TrustEmail        *bool             `json:"trustEmail"`
}

func (s SAMLSettings) signRequests() bool {
	if s.SignRequests == nil {
		return true
	}
	return *s.SignRequests
}

func (s SAMLSettings) trustEmail() bool {
	if s.TrustEmail == nil {
		return true
	}
	return *s.TrustEmail
}

func ParseSAML(raw json.RawMessage) (SAMLSettings, error) {
	var s SAMLSettings
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &s); err != nil {
			return SAMLSettings{}, err
		}
	}
	if s.AttributeMapping == nil {
		s.AttributeMapping = map[string]string{}
	}
	return s, nil
}

type metadataEntry struct {
	fetched time.Time
	meta    *saml.EntityDescriptor
}

var (
	metadataMu    sync.Mutex
	metadataCache = map[string]metadataEntry{}
)

func loadIDPMetadata(ctx context.Context, settings SAMLSettings) (*saml.EntityDescriptor, error) {
	if xml := strings.TrimSpace(settings.IDPMetadataXML); xml != "" {
		return samlsp.ParseMetadata([]byte(xml))
	}
	rawURL := strings.TrimSpace(settings.IDPMetadataURL)
	if rawURL == "" {
		return nil, ErrInvalidProvider
	}
	metadataMu.Lock()
	if hit, ok := metadataCache[rawURL]; ok && time.Since(hit.fetched) < time.Hour && hit.meta != nil {
		metadataMu.Unlock()
		return hit.meta, nil
	}
	metadataMu.Unlock()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	// The metadata URL is the trust anchor for a single IdP. Federation-aggregate
	// signature checks need the registrar's certificate and are out of scope.
	meta, err := samlsp.FetchMetadata(ctx, &http.Client{Timeout: 15 * time.Second}, *parsed)
	if err != nil {
		return nil, err
	}
	metadataMu.Lock()
	metadataCache[rawURL] = metadataEntry{fetched: time.Now(), meta: meta}
	metadataMu.Unlock()
	return meta, nil
}

func parseSigner(keyPEM string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("sp private key pem is empty")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse sp private key: %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.New("sp private key is not a signer")
	}
	return signer, nil
}

func parseCert(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("sp certificate pem is empty")
	}
	return x509.ParseCertificate(block.Bytes)
}

func serviceProvider(settings SAMLSettings, keyPEM, acs, metadataURL string, idp *saml.EntityDescriptor) (*saml.ServiceProvider, error) {
	signer, err := parseSigner(keyPEM)
	if err != nil {
		return nil, err
	}
	cert, err := parseCert(settings.SPCertPEM)
	if err != nil {
		return nil, err
	}
	acsURL, err := url.Parse(acs)
	if err != nil {
		return nil, err
	}
	mdURL, err := url.Parse(metadataURL)
	if err != nil {
		return nil, err
	}
	format := saml.PersistentNameIDFormat
	switch settings.NameIDFormat {
	case "", string(saml.PersistentNameIDFormat):
		format = saml.PersistentNameIDFormat
	case string(saml.EmailAddressNameIDFormat):
		format = saml.EmailAddressNameIDFormat
	case string(saml.UnspecifiedNameIDFormat):
		format = saml.UnspecifiedNameIDFormat
	case string(saml.TransientNameIDFormat):
		format = saml.TransientNameIDFormat
	default:
		format = saml.NameIDFormat(settings.NameIDFormat)
	}
	sp := &saml.ServiceProvider{
		EntityID:              settings.SPEntityID,
		Key:                   signer,
		Certificate:           cert,
		MetadataURL:           *mdURL,
		AcsURL:                *acsURL,
		IDPMetadata:           idp,
		AuthnNameIDFormat:     format,
		MetadataValidDuration: 365 * 24 * time.Hour,
		AllowIDPInitiated:     settings.AllowIDPInitiated,
	}
	if settings.signRequests() {
		if _, ok := signer.(*rsa.PrivateKey); ok {
			sp.SignatureMethod = dsig.RSASHA256SignatureMethod
		}
	}
	if sp.EntityID == "" {
		sp.EntityID = metadataURL
	}
	return sp, nil
}

// StartSAML builds a signed AuthnRequest redirect.
func StartSAML(ctx context.Context, p Provider, keyPEM, acs, metadataURL, returnTo, invite string) (AuthRequest, string, error) {
	settings, err := ParseSAML(p.Config)
	if err != nil {
		return AuthRequest{}, "", err
	}
	idp, err := loadIDPMetadata(ctx, settings)
	if err != nil {
		return AuthRequest{}, "", err
	}
	sp, err := serviceProvider(settings, keyPEM, acs, metadataURL, idp)
	if err != nil {
		return AuthRequest{}, "", err
	}
	authn, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return AuthRequest{}, "", err
	}
	redirect, err := authn.Redirect(authn.ID, sp)
	if err != nil {
		return AuthRequest{}, "", err
	}
	req := AuthRequest{
		ID:          authn.ID,
		ProviderID:  p.ID,
		OrgID:       p.OrgID,
		ReturnTo:    SafeReturnTo(returnTo),
		InviteToken: invite,
		ExpiresAt:   time.Now().Add(requestTTL),
	}
	return req, redirect.String(), nil
}

// FinishSAML validates the POSTed SAMLResponse against the request ID.
func FinishSAML(ctx context.Context, p Provider, keyPEM, acs, metadataURL string, r *http.Request, requestID string, allowUnsolicited bool) (Result, string, time.Time, error) {
	settings, err := ParseSAML(p.Config)
	if err != nil {
		return Result{}, "", time.Time{}, err
	}
	idp, err := loadIDPMetadata(ctx, settings)
	if err != nil {
		return Result{}, "", time.Time{}, err
	}
	sp, err := serviceProvider(settings, keyPEM, acs, metadataURL, idp)
	if err != nil {
		return Result{}, "", time.Time{}, err
	}
	var ids []string
	if requestID != "" {
		ids = []string{requestID}
	} else if !allowUnsolicited && !settings.AllowIDPInitiated {
		return Result{}, "", time.Time{}, ErrNotFound
	}
	assertion, err := sp.ParseResponse(r, ids)
	if err != nil {
		return Result{}, "", time.Time{}, err
	}
	identity, err := IdentityFromAssertion(assertion, settings)
	if err != nil {
		return Result{}, "", time.Time{}, err
	}
	expires := time.Now().Add(24 * time.Hour)
	var sessionFor time.Duration
	if assertion.Conditions != nil && !assertion.Conditions.NotOnOrAfter.IsZero() {
		expires = assertion.Conditions.NotOnOrAfter
		sessionFor = time.Until(assertion.Conditions.NotOnOrAfter)
	}
	for _, stmt := range assertion.AuthnStatements {
		if stmt.SessionNotOnOrAfter != nil && !stmt.SessionNotOnOrAfter.IsZero() {
			sessionFor = time.Until(*stmt.SessionNotOnOrAfter)
		}
	}
	return Result{Identity: identity, SessionFor: sessionFor}, assertion.ID, expires, nil
}

// IdentityFromAssertion applies eduPerson defaults and configured attribute names.
func IdentityFromAssertion(assertion *saml.Assertion, settings SAMLSettings) (Identity, error) {
	attrs := map[string][]string{}
	raw := map[string]any{}
	if assertion != nil {
		for _, stmt := range assertion.AttributeStatements {
			for _, attr := range stmt.Attributes {
				vals := make([]string, 0, len(attr.Values))
				for _, v := range attr.Values {
					if strings.TrimSpace(v.Value) != "" {
						vals = append(vals, strings.TrimSpace(v.Value))
					}
				}
				if attr.Name != "" {
					attrs[attr.Name] = append(attrs[attr.Name], vals...)
					raw[attr.Name] = attrs[attr.Name]
				}
				if attr.FriendlyName != "" && attr.FriendlyName != attr.Name {
					attrs[attr.FriendlyName] = append(attrs[attr.FriendlyName], vals...)
				}
			}
		}
	}

	var nameID, nameFormat string
	if assertion != nil && assertion.Subject != nil && assertion.Subject.NameID != nil {
		nameID = strings.TrimSpace(assertion.Subject.NameID.Value)
		nameFormat = assertion.Subject.NameID.Format
		raw["nameID"] = nameID
		raw["nameIDFormat"] = nameFormat
	}

	subject := firstAttr(attrs, settings.AttributeMapping["subject"])
	if subject == "" {
		subject = firstAttr(attrs, attrSubjectID, attrPairwiseID, attrEPPN, "eduPersonPrincipalName")
	}
	if subject == "" && nameID != "" && nameFormat != string(saml.TransientNameIDFormat) {
		subject = nameID
	}
	if subject == "" {
		return Identity{}, ErrTransientSubject
	}

	email := firstAttr(attrs, settings.AttributeMapping["email"])
	if email == "" {
		email = firstAttr(attrs, attrMail, "mail", "email")
	}
	if email == "" && nameFormat == string(saml.EmailAddressNameIDFormat) {
		email = nameID
	}

	display := firstAttr(attrs, settings.AttributeMapping["displayName"])
	if display == "" {
		display = firstAttr(attrs, attrDisplayName, "displayName")
	}
	if display == "" {
		given := firstAttr(attrs, attrGivenName, "givenName")
		sn := firstAttr(attrs, attrSurname, "sn")
		display = strings.TrimSpace(given + " " + sn)
	}

	var groups []string
	if mapped := settings.AttributeMapping["groups"]; mapped != "" {
		groups = attrs[mapped]
	} else {
		for _, key := range []string{attrEntitlement, attrIsMemberOf, attrScopedAffiliation, attrAffiliation, "eduPersonEntitlement", "isMemberOf", "eduPersonScopedAffiliation", "eduPersonAffiliation"} {
			groups = appendUnique(groups, attrs[key]...)
		}
	}

	return Identity{
		Subject:       subject,
		Email:         email,
		EmailVerified: settings.trustEmail() && email != "",
		DisplayName:   display,
		Groups:        groups,
		Raw:           raw,
	}, nil
}

func firstAttr(attrs map[string][]string, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if vals := attrs[key]; len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return ""
}

// MetadataXML returns SP metadata with POST ACS only and eduPerson requested attributes.
func MetadataXML(settings SAMLSettings, keyPEM, acs, metadataURL string) ([]byte, error) {
	sp, err := serviceProvider(settings, keyPEM, acs, metadataURL, nil)
	if err != nil {
		return nil, err
	}
	md := sp.Metadata()
	required := true
	optional := false
	reqAttr := func(name, friendly string, isRequired bool) saml.RequestedAttribute {
		flag := optional
		if isRequired {
			flag = required
		}
		return saml.RequestedAttribute{
			Attribute: saml.Attribute{
				Name:         name,
				FriendlyName: friendly,
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			},
			IsRequired: &flag,
		}
	}
	if len(md.SPSSODescriptors) > 0 {
		desc := &md.SPSSODescriptors[0]
		kept := desc.AssertionConsumerServices[:0]
		for _, acsEp := range desc.AssertionConsumerServices {
			if acsEp.Binding == saml.HTTPPostBinding {
				kept = append(kept, acsEp)
			}
		}
		desc.AssertionConsumerServices = kept
		desc.NameIDFormats = []saml.NameIDFormat{
			saml.PersistentNameIDFormat,
			saml.TransientNameIDFormat,
			saml.UnspecifiedNameIDFormat,
		}
		desc.AttributeConsumingServices = []saml.AttributeConsumingService{{
			Index:        1,
			IsDefault:    &required,
			ServiceNames: []saml.LocalizedName{{Lang: "en", Value: "LRSolid"}},
			RequestedAttributes: []saml.RequestedAttribute{
				reqAttr(attrSubjectID, "subject-id", true),
				reqAttr(attrEPPN, "eduPersonPrincipalName", true),
				reqAttr(attrMail, "mail", true),
				reqAttr(attrDisplayName, "displayName", false),
				reqAttr(attrGivenName, "givenName", false),
				reqAttr(attrSurname, "sn", false),
				reqAttr(attrScopedAffiliation, "eduPersonScopedAffiliation", false),
				reqAttr(attrEntitlement, "eduPersonEntitlement", false),
				reqAttr(attrIsMemberOf, "isMemberOf", false),
			},
		}}
	}
	return xmlMarshal(md)
}

func xmlMarshal(v any) ([]byte, error) {
	return marshalMetadata(v)
}

// TestSAML fetches or parses IdP metadata and reports the entityID.
func TestSAML(ctx context.Context, raw json.RawMessage) (string, error) {
	settings, err := ParseSAML(raw)
	if err != nil {
		return "", err
	}
	meta, err := loadIDPMetadata(ctx, settings)
	if err != nil {
		return "", err
	}
	return "idp " + meta.EntityID, nil
}
