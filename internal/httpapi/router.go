package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
)

func NewRouter(cfg config.Config, svc *lrs.Service, db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Heartbeat("/healthz"))

	// Health
	r.Get("/healthz", Health())

	// xAPI minimal
	r.Route("/xapi", func(r chi.Router) {
		r.Route("/statements", func(r chi.Router) {
			r.Post("/", PostStatements(cfg, svc))
			r.Get("/", GetStatements(cfg, svc))
		})
	})

	// Admin (web UI) API
	r.Route("/api", func(r chi.Router) {
		r.Route("/admin", func(r chi.Router) {
			r.Get("/dashboard", AdminDashboard(cfg, db))
			r.Get("/statements", AdminListStatements(cfg, svc))
			r.Get("/statements/{id}", AdminGetStatement(cfg, svc))
			r.Get("/users", AdminListUsers(cfg, db))
			r.Post("/users", AdminCreateUser(cfg, db))
			r.Patch("/users/{id}", AdminUpdateUser(cfg, db))
		})
	})

	return r
}
