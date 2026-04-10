package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
)

const sessionTTL = 24 * time.Hour

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authMeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Login handles POST /api/auth/login.
// On success it sets an httpOnly session cookie containing a signed JWT.
func Login(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "email and password required"})
			return
		}

		ctx := r.Context()

		var userID, email, role string
		var passwordHash *string
		err := db.QueryRow(ctx, `
			SELECT u.id, u.email, om.role, u.password_hash
			FROM users u
			JOIN org_memberships om ON om.user_id = u.id
			JOIN organisations org   ON org.id = om.org_id
			WHERE u.email = $1
			  AND org.slug = $2
			  AND u.disabled_at IS NULL
			LIMIT 1
		`, req.Email, cfg.DefaultOrg).Scan(&userID, &email, &role, &passwordHash)
		if err != nil || passwordHash == nil {
			// Same response for "not found" and "no password" to prevent user enumeration
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(req.Password)); err != nil {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
			return
		}

		token, err := auth.GenerateToken(cfg.JWTSecret, auth.Claims{
			UserID: userID,
			Email:  email,
			Role:   role,
		}, sessionTTL)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "token generation failed"})
			return
		}

		http.SetCookie(w, sessionCookie(token, int(sessionTTL.Seconds()), cfg.Environment != "dev"))
		writeJSON(w, http.StatusOK, authMeResponse{ID: userID, Email: email, Role: role})
	}
}

// Logout handles POST /api/auth/logout.
// Clears the session cookie.
func Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, sessionCookie("", -1, false))
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Me handles GET /api/auth/me.
// Returns the current user from the validated JWT claims (set by auth.Middleware).
func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
			return
		}
		writeJSON(w, http.StatusOK, authMeResponse{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		})
	}
}

func sessionCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
