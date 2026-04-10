package auth

import (
	"context"
	"net/http"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

const SessionCookie = "lrsolid_session"

// Middleware validates the session cookie and injects Claims into the request context.
// Returns 401 if the cookie is missing, invalid, or expired.
func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookie)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, err := ValidateToken(jwtSecret, cookie.Value)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

// ClaimsFromContext retrieves auth Claims from a request context set by Middleware.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}
