package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/acme/observability/internal/agenthub"
	"github.com/acme/observability/internal/alerting"
	"github.com/acme/observability/internal/analyzer"
	"github.com/acme/observability/internal/config"
	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/handlers"
	"github.com/acme/observability/internal/migrate"
	"github.com/acme/observability/internal/remote"
	chrepo "github.com/acme/observability/internal/repositories/clickhouse"
	pgrepo "github.com/acme/observability/internal/repositories/postgres"
	"github.com/acme/observability/internal/ws"
	"github.com/acme/observability/pkg/logger"
	"github.com/acme/observability/pkg/notify"
	"github.com/acme/observability/pkg/secrets"
)

func main() {
	log := logger.New()
	cfg := config.Load()

	ctx := context.Background()

	// --- Databases ---
	pg, err := pgrepo.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pg.Close()
	log.Info("connected to postgres")

	ch, err := chrepo.Connect(ctx, cfg.ClickHouseDSN)
	if err != nil {
		log.Error("clickhouse connect failed", "err", err)
		os.Exit(1)
	}
	defer ch.Close()
	log.Info("connected to clickhouse")

	// --- Schema migrations (self-applied, idempotent) ---
	if err := migrate.Postgres(ctx, pg, cfg.SeedDemo, log); err != nil {
		log.Error("postgres migrate failed", "err", err)
		os.Exit(1)
	}
	if err := migrate.ClickHouse(ctx, ch, cfg.SeedDemo, log); err != nil {
		log.Error("clickhouse migrate failed", "err", err)
		os.Exit(1)
	}
	log.Info("migrations applied", "seed_demo", cfg.SeedDemo)

	// --- Repositories ---
	userRepo := pgrepo.NewUserRepo(pg)
	orgRepo := pgrepo.NewOrgRepo(pg)
	projectRepo := pgrepo.NewProjectRepo(pg)
	serviceRepo := pgrepo.NewServiceRepo(pg)
	deploymentRepo := pgrepo.NewDeploymentRepo(pg)
	alertRepo := pgrepo.NewAlertRepo(pg)
	timelineRepo := pgrepo.NewTimelineRepo(pg)
	ingestKeyRepo := pgrepo.NewIngestKeyRepo(pg)
	alertRuleRepo := pgrepo.NewAlertRuleRepo(pg)
	serverRepo := pgrepo.NewServerRepo(pg)
	auditRepo := pgrepo.NewAuditRepo(pg)
	metricRepo := chrepo.NewMetricRepo(ch)
	logRepo := chrepo.NewLogRepo(ch)

	// --- Realtime + agent control hubs ---
	hub := ws.NewHub()
	agentControl := agenthub.New()

	// --- Services (dependency injection) ---
	tokens := services.NewTokenService(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.AccessTTL, cfg.RefreshTTL)
	authService := services.NewAuthService(userRepo, tokens)
	coreService := &services.CoreService{
		Orgs:        orgRepo,
		Projects:    projectRepo,
		Services:    serviceRepo,
		Deployments: deploymentRepo,
		Alerts:      alertRepo,
		Timeline:    timelineRepo,
		Metrics:     metricRepo,
		Logs:        logRepo,
		IngestKeys:      ingestKeyRepo,
		AlertRules:      alertRuleRepo,
		Servers:         serverRepo,
		Audit:           auditRepo,
		Analyzer:        analyzer.NewDeterministic(),
		Hub:             hub,
		Exec:            remote.NewSSH(),
		Secrets:         secrets.New(cfg.CredentialsKey),
		PublicIngestURL: cfg.PublicIngestURL,
	}

	// --- Alert evaluator (background) ---
	evaluator := &alerting.Evaluator{
		Rules:    alertRuleRepo,
		Metrics:  metricRepo,
		Alerts:   alertRepo,
		Timeline: timelineRepo,
		Services: serviceRepo,
		Hub:      hub,
		Notifier: notify.New(),
		Interval: 15 * time.Second,
		Window:   2 * time.Minute,
		Log:      log,
	}
	evalCtx, stopEval := context.WithCancel(context.Background())
	defer stopEval()
	go evaluator.Run(evalCtx)

	// --- Handlers + Router ---
	authHandler := handlers.NewAuthHandler(authService)
	coreHandler := handlers.NewCoreHandler(coreService, hub, agentControl)
	ingestHandler := handlers.NewIngestHandler(coreService)
	router := handlers.NewRouter(cfg, tokens, ingestKeyRepo, authHandler, coreHandler, ingestHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
