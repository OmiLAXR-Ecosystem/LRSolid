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

	// Docs (OpenAPI)
	r.Get("/docs", SwaggerUI())
	r.Get("/docs/openapi.yaml", OpenAPISpec())

	// Connection API (LL-inspired)
	r.Route("/api", func(r chi.Router) {
		r.Route("/connection", func(r chi.Router) {
			r.Get("/statement", ConnectionStatements(cfg, svc))
		})
		r.Get("/statement/{id}", StatementByID(cfg, svc))

		// REST API (LL-inspired)
		r.Route("/v1", func(r chi.Router) {
			r.Get("/user", RestListUsers(cfg, db))
			r.Get("/user/count", RestCountUsers(cfg, db))
		})

		// Aggregation API (LL-inspired)
		r.Route("/aggregation", func(r chi.Router) {
			r.Get("/statement", AggregationStatementsNotImplemented())
			r.Post("/statement", AggregationStatementsNotImplemented())
		})
	})

	return r
}
