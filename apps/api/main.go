package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/datastore"
	"github.com/SGoerzen/LRSolid/internal/httpapi"
	"github.com/SGoerzen/LRSolid/internal/limits"
	"github.com/SGoerzen/LRSolid/internal/lrs"
	"github.com/SGoerzen/LRSolid/internal/plugins"
	"github.com/SGoerzen/LRSolid/internal/plugins/builtin"
	"github.com/SGoerzen/LRSolid/internal/plugins/jsplugin"
	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/sso"
	"github.com/SGoerzen/LRSolid/internal/statsrollup"
	"github.com/SGoerzen/LRSolid/internal/store"
	mgstore "github.com/SGoerzen/LRSolid/internal/store/mongodb"
	pgstore "github.com/SGoerzen/LRSolid/internal/store/postgres"
)

func runMigrations(dsn string) {
	// golang-migrate pgx/v5 driver expects pgx5:// scheme
	pgxDSN := "pgx5://" + dsn[len("postgres://"):]
	m, err := migrate.New("file://migrations", pgxDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("migrate: init failed")
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal().Err(err).Msg("migrate: up failed")
	}
	log.Info().Msg("migrate: schema up to date")
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config load failed")
	}

	runMigrations(cfg.PostgresDSN)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgstore.Open(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres connect failed")
	}
	defer db.Close()

	storeRegistry := store.NewRegistry()
	storeRegistry.RegisterFactory("postgres", pgstore.NewStoreBundle)
	storeRegistry.RegisterFactory("mongodb", mgstore.NewStoreBundle)

	var dataSecretStore secrets.Store
	if cfg.DataStoreSecretKey != "" {
		dataSecretStore, err = secrets.NewPostgresStore(db, cfg.DataStoreSecretKey)
		if err != nil {
			log.Fatal().Err(err).Msg("data store secret store init failed")
		}
	} else {
		log.Warn().Msg("DATA_STORE_SECRET_KEY not set; encrypted external data store credentials are unavailable")
	}

	reconcileStores := func() {
		cfgs, err := datastore.LoadActive(ctx, db, dataSecretStore)
		if err != nil {
			log.Error().Err(err).Msg("load data stores failed")
			return
		}
		for _, ev := range storeRegistry.Reconcile(ctx, cfgs) {
			log.Warn().Err(ev.Err).Str("name", ev.Name).Msg("data store registration failed")
		}
	}
	reconcileStores()
	for _, id := range storeRegistry.IDs() {
		log.Info().Str("id", id).Msg("data store registered")
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcileStores()
			}
		}
	}()

	// Default fallback: the control-plane Postgres (no connection_string needed).
	fallbackStmtStore := pgstore.NewStatementStore(db)
	svc := lrs.NewService(storeRegistry, fallbackStmtStore, statsrollup.Counters{DB: db}, limits.Checker{DB: db})

	registry := plugins.Global
	builtin.Register(registry)

	if cfg.PluginsDir != "" {
		n, err := jsplugin.LoadDir(cfg.PluginsDir, registry)
		if err != nil {
			log.Fatal().Err(err).Msg("js plugins load failed")
		}
		log.Info().Int("count", n).Str("dir", cfg.PluginsDir).Msg("js plugins loaded")
	}

	if err := sso.Seed(ctx, db, dataSecretStore, cfg); err != nil {
		log.Error().Err(err).Msg("sso seed failed")
	}

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sso.Cleanup(ctx, db); err != nil {
					log.Error().Err(err).Msg("sso cleanup failed")
				}
			}
		}
	}()

	router := httpapi.NewRouter(cfg, svc, db, registry, dataSecretStore, storeRegistry)

	if cfg.StatsRollupEnabled {
		worker := statsrollup.New(db)
		go func() {
			interval := time.Duration(cfg.StatsRollupIntervalS) * time.Second
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			run := func() {
				// Catch up in a few bursts per tick if backlog exists.
				for i := 0; i < 10; i++ {
					n, err := worker.RunBatch(ctx, cfg.StatsRollupBatchSize)
					if err != nil {
						log.Error().Err(err).Msg("stats rollup batch failed")
						return
					}
					if n < cfg.StatsRollupBatchSize {
						return
					}
				}
			}

			run()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					run()
				}
			}
		}()
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("api listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	storeRegistry.Close()
}
