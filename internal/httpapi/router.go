package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
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

	// Public routes
	r.Get("/healthz", Health())
	r.Get("/docs", SwaggerUI())
	r.Get("/docs/openapi.yaml", OpenAPISpec())

	// Auth (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", Login(cfg, db))
		r.Post("/logout", Logout())
	})

	// Protected routes — require valid session cookie
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		// Auth: current user
		r.Get("/api/auth/me", Me())

		// xAPI
		r.Route("/xapi", func(r chi.Router) {
			r.Route("/statements", func(r chi.Router) {
				r.Post("/", PostStatements(cfg, svc))
				r.Get("/", GetStatements(cfg, svc))
			})
		})

		// Connection + REST API
		r.Route("/api", func(r chi.Router) {
			r.Route("/connection", func(r chi.Router) {
				r.Get("/statement", ConnectionStatements(cfg, svc))
			})
			r.Get("/statement/{id}", StatementByID(cfg, svc))

			r.Route("/v1", func(r chi.Router) {
				r.Get("/user", RestListUsers(cfg, db))
				r.Get("/user/count", RestCountUsers(cfg, db))
			})

			r.Route("/aggregation", func(r chi.Router) {
				r.Get("/statement", AggregationStatementsNotImplemented())
				r.Post("/statement", AggregationStatementsNotImplemented())
			})
		})
	})

	return r
}
