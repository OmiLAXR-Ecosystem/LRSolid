package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/sso"
)

type loginProvidersResponse struct {
	LocalLoginEnabled bool                `json:"localLoginEnabled"`
	Providers         []sso.LoginProvider `json:"providers"`
}

func ListLoginProviders(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org := strings.TrimSpace(r.URL.Query().Get("org"))
		if org == "" {
			org = cfg.DefaultOrg
		}
		providers, err := sso.ListLoginProviders(r.Context(), db, org)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list providers failed"})
			return
		}
		enabled, err := sso.LocalLoginEnabled(r.Context(), db, org)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list providers failed"})
			return
		}
		writeJSON(w, http.StatusOK, loginProvidersResponse{LocalLoginEnabled: enabled, Providers: providers})
	}
}

func SSOStart(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ssoLimiter.allow(r.RemoteAddr) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		slug := chi.URLParam(r, "slug")
		org := strings.TrimSpace(r.URL.Query().Get("org"))
		p, err := sso.LoadBySlug(r.Context(), db, slug, org)
		if err != nil {
			redirectLoginError(w, r, "unknown_provider")
			return
		}
		secret, err := providerSecret(r.Context(), store, p)
		if err != nil {
			log.Error().Err(err).Str("provider", p.Slug).Msg("sso secret load failed")
			redirectLoginError(w, r, "provider_error")
			return
		}
		returnTo := r.URL.Query().Get("return_to")
		invite := strings.TrimSpace(r.URL.Query().Get("invite"))
		var req sso.AuthRequest
		var redirect string
		switch p.Type {
		case sso.TypeOIDC:
			req, redirect, err = sso.StartOIDC(r.Context(), p, secret, sso.OIDCRedirectURL(cfg.AppURL, p.Slug), returnTo, invite)
		case sso.TypeSAML:
			req, redirect, err = sso.StartSAML(r.Context(), p, secret, sso.SAMLACSURL(cfg.AppURL, p.Slug), sso.SAMLMetadataURL(cfg.AppURL, p.Slug), returnTo, invite)
		default:
			err = sso.ErrInvalidProvider
		}
		if err != nil {
			log.Warn().Err(err).Str("provider", p.Slug).Msg("sso start failed")
			redirectLoginError(w, r, "provider_error")
			return
		}
		orgID, err := startOrgID(r.Context(), db, cfg, p, org)
		if err != nil {
			redirectLoginError(w, r, "provider_error")
			return
		}
		req.OrgID = &orgID
		if err := sso.SaveRequest(r.Context(), db, req); err != nil {
			log.Error().Err(err).Str("provider", p.Slug).Msg("sso request save failed")
			redirectLoginError(w, r, "provider_error")
			return
		}
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

func OIDCCallback(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ssoLimiter.allow(r.RemoteAddr) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		if r.URL.Query().Get("error") != "" {
			redirectLoginError(w, r, "provider_error")
			return
		}
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		req, err := sso.TakeRequest(r.Context(), db, state)
		if err != nil {
			redirectLoginError(w, r, loginErrorCode(err))
			return
		}
		p, err := sso.LoadByID(r.Context(), db, req.ProviderID)
		if err != nil || !p.Enabled || p.Type != sso.TypeOIDC || p.Slug != chi.URLParam(r, "slug") {
			redirectLoginError(w, r, "provider_error")
			return
		}
		secret, err := providerSecret(r.Context(), store, p)
		if err != nil {
			redirectLoginError(w, r, "provider_error")
			return
		}
		result, err := sso.FinishOIDC(r.Context(), p, secret, sso.OIDCRedirectURL(cfg.AppURL, p.Slug), req, code)
		if err != nil {
			log.Warn().Err(err).Str("provider", p.Slug).Msg("oidc callback failed")
			redirectLoginError(w, r, loginErrorCode(err))
			return
		}
		completeSSOLogin(w, r, cfg, db, p, req, result)
	}
}

func SAMLACS(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ssoLimiter.allow(r.RemoteAddr) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := r.ParseForm(); err != nil {
			redirectLoginError(w, r, "provider_error")
			return
		}
		slug := chi.URLParam(r, "slug")
		p, err := sso.LoadBySlug(r.Context(), db, slug, "")
		if err != nil {
			redirectLoginError(w, r, "unknown_provider")
			return
		}
		settings, err := sso.ParseSAML(p.Config)
		if err != nil {
			redirectLoginError(w, r, "provider_error")
			return
		}
		relay := r.FormValue("RelayState")
		var req sso.AuthRequest
		if relay != "" {
			req, err = sso.TakeRequest(r.Context(), db, relay)
			if err != nil {
				redirectLoginError(w, r, loginErrorCode(err))
				return
			}
			if req.ProviderID != p.ID {
				redirectLoginError(w, r, "provider_error")
				return
			}
		} else if !settings.AllowIDPInitiated {
			redirectLoginError(w, r, "provider_error")
			return
		}
		secret, err := providerSecret(r.Context(), store, p)
		if err != nil {
			redirectLoginError(w, r, "provider_error")
			return
		}
		result, assertionID, assertionExp, err := sso.FinishSAML(
			r.Context(), p, secret,
			sso.SAMLACSURL(cfg.AppURL, p.Slug),
			sso.SAMLMetadataURL(cfg.AppURL, p.Slug),
			r, req.ID, settings.AllowIDPInitiated,
		)
		if err != nil {
			log.Warn().Err(err).Str("provider", p.Slug).Msg("saml acs failed")
			redirectLoginError(w, r, loginErrorCode(err))
			return
		}
		if err := sso.SeeAssertion(r.Context(), db, assertionID, assertionExp); err != nil {
			redirectLoginError(w, r, loginErrorCode(err))
			return
		}
		if req.OrgID == nil {
			orgID, err := startOrgID(r.Context(), db, cfg, p, "")
			if err != nil {
				redirectLoginError(w, r, "provider_error")
				return
			}
			req.OrgID = &orgID
		}
		completeSSOLogin(w, r, cfg, db, p, req, result)
	}
}

func SAMLMetadata(cfg config.Config, db *pgxpool.Pool, store secrets.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := sso.LoadAnyBySlug(r.Context(), db, chi.URLParam(r, "slug"))
		if err != nil || p.Type != sso.TypeSAML {
			http.NotFound(w, r)
			return
		}
		settings, err := sso.ParseSAML(p.Config)
		if err != nil {
			http.Error(w, "invalid provider", http.StatusInternalServerError)
			return
		}
		secret, err := providerSecret(r.Context(), store, p)
		if err != nil {
			http.Error(w, "secret unavailable", http.StatusInternalServerError)
			return
		}
		body, err := sso.MetadataXML(settings, secret, sso.SAMLACSURL(cfg.AppURL, p.Slug), sso.SAMLMetadataURL(cfg.AppURL, p.Slug))
		if err != nil {
			http.Error(w, "metadata failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/samlmetadata+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func completeSSOLogin(w http.ResponseWriter, r *http.Request, cfg config.Config, db *pgxpool.Pool, p sso.Provider, req sso.AuthRequest, result sso.Result) {
	orgID := ""
	if req.OrgID != nil {
		orgID = *req.OrgID
	}
	if orgID == "" {
		redirectLoginError(w, r, "provider_error")
		return
	}
	mapping, requireVerified, err := providerPolicy(p)
	if err != nil {
		redirectLoginError(w, r, "provider_error")
		return
	}
	user, err := sso.Resolve(r.Context(), db, p, result.Identity, orgID, req.InviteToken, mapping, requireVerified)
	if err != nil {
		log.Info().Err(err).Str("provider", p.Slug).Msg("sso identity rejected")
		redirectLoginError(w, r, loginErrorCode(err))
		return
	}
	if err := issueSession(w, cfg, user, sessionDuration(result.SessionFor)); err != nil {
		redirectLoginError(w, r, "provider_error")
		return
	}
	http.Redirect(w, r, req.ReturnTo, http.StatusFound)
}

func issueSession(w http.ResponseWriter, cfg config.Config, user sso.SessionUser, ttl time.Duration) error {
	token, err := auth.GenerateToken(cfg.JWTSecret, auth.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	}, ttl)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessionCookie(token, int(ttl.Seconds()), cfg.Environment != "dev"))
	return nil
}

func sessionDuration(forDur time.Duration) time.Duration {
	if forDur <= 0 || forDur >= sessionTTL {
		return sessionTTL
	}
	if forDur < 5*time.Minute {
		return 5 * time.Minute
	}
	return forDur
}

func providerPolicy(p sso.Provider) (map[string]string, bool, error) {
	switch p.Type {
	case sso.TypeOIDC:
		settings, err := sso.ParseOIDC(p.Config)
		if err != nil {
			return nil, false, err
		}
		return settings.RoleMapping, settings.RequireEmailVerified, nil
	case sso.TypeSAML:
		settings, err := sso.ParseSAML(p.Config)
		if err != nil {
			return nil, false, err
		}
		return settings.RoleMapping, false, nil
	default:
		return nil, false, sso.ErrInvalidProvider
	}
}

func providerSecret(ctx context.Context, store secrets.Store, p sso.Provider) (string, error) {
	if p.SecretRef == nil || *p.SecretRef == "" {
		return "", nil
	}
	if store == nil {
		return "", errors.New("secret store not configured")
	}
	b, err := store.Get(ctx, *p.SecretRef)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func startOrgID(ctx context.Context, db *pgxpool.Pool, cfg config.Config, p sso.Provider, requestedOrg string) (string, error) {
	if p.OrgID != nil && *p.OrgID != "" {
		return *p.OrgID, nil
	}
	slug := requestedOrg
	if slug == "" {
		slug = cfg.DefaultOrg
	}
	id, err := getOrgID(ctx, db, slug)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", sso.ErrNotFound
	}
	return id, nil
}

func redirectLoginError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+code, http.StatusFound)
}

func loginErrorCode(err error) string {
	switch {
	case errors.Is(err, sso.ErrNoAccount):
		return "no_account"
	case errors.Is(err, sso.ErrDisabled):
		return "disabled"
	case errors.Is(err, sso.ErrEmailUnverified):
		return "email_unverified"
	case errors.Is(err, sso.ErrEmailRequired):
		return "email_required"
	case errors.Is(err, sso.ErrTransientSubject):
		return "transient_subject"
	case errors.Is(err, sso.ErrExpired), errors.Is(err, sso.ErrNotFound), errors.Is(err, sso.ErrReplay):
		return "provider_error"
	default:
		return "provider_error"
	}
}

type slidingLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (l *slidingLimiter) allow(key string) bool {
	const limit = 30
	window := time.Minute
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hits == nil {
		l.hits = map[string][]time.Time{}
	}
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

var ssoLimiter slidingLimiter
