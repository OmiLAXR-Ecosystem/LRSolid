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
	r.Use(middleware.StripSlashes)
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

	// xAPI routes: require API client credentials (HTTP Basic key_id:secret)
	r.Route("/xapi", func(r chi.Router) {
		r.Use(XAPIClientAuth(cfg, db))
		r.Post("/statements", PostStatements(cfg, svc))
		r.Get("/statements", GetStatements(cfg, svc))
	})

	// Protected routes — require valid session cookie
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		// Auth: current user
		r.Get("/api/auth/me", Me())

		// Connection + REST API
		r.Route("/api", func(r chi.Router) {
			r.Route("/connection", func(r chi.Router) {
				r.Get("/statement", ConnectionStatements(cfg, svc))
			})
			r.Get("/statement/{id}", StatementByID(cfg, svc))

			r.Route("/v1", func(r chi.Router) {
				// Organisation list (no org param required)
				r.Get("/org", RestListOrgs(cfg, db))

				// Org-scoped endpoints
				r.Route("/{org}", func(r chi.Router) {
					r.Get("/dashboard", AdminDashboard(cfg, db))
					r.Get("/user", RestListUsers(cfg, db))
					r.Get("/user/count", RestCountUsers(cfg, db))
					r.Get("/project", RestListProjects(cfg, db))
					r.Post("/project", RestCreateProject(cfg, db))
					r.Delete("/project/{id}", RestDeleteProject(cfg, db))
					r.Get("/lrs", RestListLRS(cfg, db))
					r.Post("/lrs", RestCreateLRS(cfg, db))
					r.Delete("/lrs/{id}", RestDeleteLRS(cfg, db))
					r.Get("/data-store", RestListDataStores(cfg, db))
					r.Get("/client", RestListClients(cfg, db))
					r.Post("/client", RestCreateClient(cfg, db))
					r.Delete("/client/{id}", RestDeleteClient(cfg, db))
				})
			})

			r.Route("/aggregation", func(r chi.Router) {
				r.Get("/statement", AggregationStatementsNotImplemented())
				r.Post("/statement", AggregationStatementsNotImplemented())
			})
		})
	})

	return r
}
