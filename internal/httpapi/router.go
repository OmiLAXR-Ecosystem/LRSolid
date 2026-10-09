package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/auth"
	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/plugins"
	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/store"
)

func NewRouter(cfg config.Config, svc *lrs.Service, db *pgxpool.Pool, registry *plugins.Registry, dataSecrets secrets.Store, stores *store.Registry) http.Handler {
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
		r.Get("/providers", ListLoginProviders(cfg, db))
		r.Get("/invite", GetInvite(db))
		r.Post("/invite/accept", AcceptInvite(cfg, db))
		r.Get("/sso/{slug}/start", SSOStart(cfg, db, dataSecrets))
		r.Get("/oidc/{slug}/callback", OIDCCallback(cfg, db, dataSecrets))
		r.Post("/saml/{slug}/acs", SAMLACS(cfg, db, dataSecrets))
		r.Get("/saml/{slug}/metadata", SAMLMetadata(cfg, db, dataSecrets))
	})

	// xAPI routes. About is unauthenticated. Everything else needs client credentials.
	r.Route("/xapi", func(r chi.Router) {
		r.Use(AlternateRequestSyntax)
		r.Get("/about", GetAbout())
		r.Group(func(r chi.Router) {
			r.Use(XAPIClientAuth(cfg, db))
			r.Use(XAPIVersion(cfg))
			r.Post("/statements", PostStatements(cfg, svc))
			r.Put("/statements", PutStatements(cfg, svc))
			r.Get("/statements", GetStatements(cfg, svc))
			r.Method(http.MethodGet, "/activities/state", StateDocument(cfg, svc))
			r.Method(http.MethodPut, "/activities/state", StateDocument(cfg, svc))
			r.Method(http.MethodPost, "/activities/state", StateDocument(cfg, svc))
			r.Method(http.MethodDelete, "/activities/state", StateDocument(cfg, svc))
			r.Method(http.MethodGet, "/activities/profile", ActivityProfileDocument(cfg, svc))
			r.Method(http.MethodPut, "/activities/profile", ActivityProfileDocument(cfg, svc))
			r.Method(http.MethodPost, "/activities/profile", ActivityProfileDocument(cfg, svc))
			r.Method(http.MethodDelete, "/activities/profile", ActivityProfileDocument(cfg, svc))
			r.Method(http.MethodGet, "/agents/profile", AgentProfileDocument(cfg, svc))
			r.Method(http.MethodPut, "/agents/profile", AgentProfileDocument(cfg, svc))
			r.Method(http.MethodPost, "/agents/profile", AgentProfileDocument(cfg, svc))
			r.Method(http.MethodDelete, "/agents/profile", AgentProfileDocument(cfg, svc))
			r.Get("/activities", GetActivity(cfg, svc))
			r.Get("/agents", GetAgent(cfg, svc))
		})
	})

	// Protected routes — require valid session cookie
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		// Auth: current user
		r.Get("/api/auth/me", Me(db))

		// Connection + REST API
		r.Route("/api", func(r chi.Router) {
			r.Route("/connection", func(r chi.Router) {
				r.Get("/statement", ConnectionStatements(cfg, svc, db))
			})
			r.Get("/statement/{id}", StatementByID(cfg, svc, db))

			r.Route("/v1", func(r chi.Router) {
				// Organisation list (no org param required)
				r.Get("/org", RestListOrgs(cfg, db))
				r.Route("/admin", func(r chi.Router) {
					r.Post("/org", AdminCreateOrg(cfg, db))
					r.Post("/user", AdminCreateUser(cfg, db))
					r.Post("/user/assign", AdminAssignUserToOrg(cfg, db))
					r.Post("/data-store/test", AdminTestDataStore(cfg, db, dataSecrets, stores))
					r.Post("/data-store", AdminCreateDataStore(cfg, db, dataSecrets, stores))
					r.Post("/data-store/{id}/test", AdminTestDataStore(cfg, db, dataSecrets, stores))
					r.Patch("/data-store/{id}", AdminUpdateDataStore(cfg, db, dataSecrets, stores))
					r.Delete("/data-store/{id}", AdminDeleteDataStore(cfg, db, dataSecrets, stores))
				})

				// Org-scoped endpoints
				r.Route("/{org}", func(r chi.Router) {
					r.Get("/dashboard", AdminDashboard(cfg, db, registry))
					r.Get("/dashboard/config", RestGetDashboardConfig(cfg, db, registry))
					r.Put("/dashboard/config", RestPutDashboardConfig(cfg, db, registry))
					r.Delete("/dashboard/config", RestDeleteDashboardConfig(cfg, db))
					r.Get("/export/settings", RestGetExportSettings(cfg, db))
					r.Put("/export/settings", RestUpdateExportSettings(cfg, db))
					r.Get("/export/jobs", RestListExportJobs(cfg, db))
					r.Get("/export/jobs/{id}", RestGetExportJob(cfg, db))
					r.Get("/export/jobs/{id}/download", RestDownloadExportJob(cfg, db))
					r.Post("/export/statements", RestCreateStatementExport(cfg, db, stores))
					r.Get("/plugins", RestListPlugins(cfg, db, registry))
					r.Get("/auth-settings", GetAuthSettings(db))
					r.Put("/auth-settings", PutAuthSettings(db))
					r.Get("/auth-provider", ListAuthProviders(db))
					r.Post("/auth-provider", CreateAuthProvider(cfg, db, dataSecrets))
					r.Patch("/auth-provider/{id}", UpdateAuthProvider(cfg, db, dataSecrets))
					r.Delete("/auth-provider/{id}", DeleteAuthProvider(db, dataSecrets))
					r.Post("/auth-provider/{id}/test", TestAuthProvider(db))
					r.Get("/user", RestListUsers(cfg, db))
					r.Get("/user/count", RestCountUsers(cfg, db))
					r.Post("/user/invite", RestInviteUser(cfg, db))
					r.Get("/project", RestListProjects(cfg, db))
					r.Post("/project", RestCreateProject(cfg, db))
					r.Delete("/project/{id}", RestDeleteProject(cfg, db))
					r.Get("/lrs", RestListLRS(cfg, db))
					r.Post("/lrs", RestCreateLRS(cfg, db))
					r.Delete("/lrs/{id}", RestDeleteLRS(cfg, db))
					r.Get("/data-store", RestListDataStores(cfg, db, stores))
					r.Get("/client", RestListClients(cfg, db))
					r.Post("/client", RestCreateClient(cfg, db))
					r.Delete("/client/{id}", RestDeleteClient(cfg, db))
					r.Get("/limits", RestGetLimits(cfg, db))
					r.Put("/limits", RestPutLimit(cfg, db))
					r.Delete("/limits/{id}", RestDeleteLimit(cfg, db))
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
