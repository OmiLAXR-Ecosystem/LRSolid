package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/sso"
)

type inviteInfoResponse struct {
	Email   string    `json:"email"`
	OrgSlug string    `json:"orgSlug"`
	OrgName string    `json:"orgName"`
	Role    string    `json:"role"`
	Expires time.Time `json:"expiresAt"`
}

type acceptInviteRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func GetInvite(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info, _, _, ok := loadInvite(w, r, db, r.URL.Query().Get("token"))
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func AcceptInvite(cfg config.Config, db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req acceptInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Password = strings.TrimSpace(req.Password)
		req.DisplayName = strings.TrimSpace(req.DisplayName)
		if len(req.Password) < 8 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "password must be at least 8 characters"})
			return
		}
		info, inviteID, orgID, ok := loadInvite(w, r, db, req.Token)
		if !ok {
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "password hash failed"})
			return
		}

		tx, err := db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}
		defer func() { _ = tx.Rollback(r.Context()) }()

		var userID string
		var existingHash *string
		err = tx.QueryRow(r.Context(), `SELECT id::text, password_hash FROM users WHERE lower(email) = lower($1)`, info.Email).Scan(&userID, &existingHash)
		if errors.Is(err, pgx.ErrNoRows) {
			var display *string
			if req.DisplayName != "" {
				display = &req.DisplayName
			}
			if err := tx.QueryRow(r.Context(), `
				INSERT INTO users (email, display_name, password_hash) VALUES ($1, $2, $3) RETURNING id::text
			`, info.Email, display, string(hash)).Scan(&userID); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
				return
			}
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		} else if existingHash != nil {
			if bcrypt.CompareHashAndPassword([]byte(*existingHash), []byte(req.Password)) != nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
				return
			}
		} else if _, err := tx.Exec(r.Context(), `UPDATE users SET password_hash = $2 WHERE id = $1::uuid`, userID, string(hash)); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}

		if _, err := tx.Exec(r.Context(), `
			INSERT INTO org_memberships (org_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3)
			ON CONFLICT (org_id, user_id) DO NOTHING
		`, orgID, userID, info.Role); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}
		if _, err := tx.Exec(r.Context(), `UPDATE user_invites SET accepted_at = now() WHERE id = $1::uuid`, inviteID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}
		var role string
		if err := tx.QueryRow(r.Context(), `
			SELECT role FROM org_memberships WHERE org_id = $1::uuid AND user_id = $2::uuid
		`, orgID, userID).Scan(&role); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "accept invite failed"})
			return
		}
		if err := issueSession(w, cfg, sso.SessionUser{ID: userID, Email: info.Email, Role: role}, sessionTTL); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "token generation failed"})
			return
		}
		writeJSON(w, http.StatusOK, authMeResponse{ID: userID, Email: info.Email, Role: role})
	}
}

func loadInvite(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, token string) (inviteInfoResponse, string, string, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "token required"})
		return inviteInfoResponse{}, "", "", false
	}
	var info inviteInfoResponse
	var id, orgID string
	err := db.QueryRow(r.Context(), `
		SELECT i.id::text, i.org_id::text, i.email, i.role, i.expires_at, o.slug, o.name
		FROM user_invites i
		JOIN organisations o ON o.id = i.org_id
		WHERE i.token = $1 AND i.accepted_at IS NULL AND i.expires_at > now()
	`, token).Scan(&id, &orgID, &info.Email, &info.Role, &info.Expires, &info.OrgSlug, &info.OrgName)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "invite not found"})
		return inviteInfoResponse{}, "", "", false
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load invite failed"})
		return inviteInfoResponse{}, "", "", false
	}
	return info, id, orgID, true
}
