package handlers

import (
	"net/http"
	"time"

	"github.com/acme/observability/internal/config"
	"github.com/acme/observability/internal/domain/repositories"
	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter wires all routes, middleware and handlers.
func NewRouter(
	cfg *config.Config,
	tokens *services.TokenService,
	keys repositories.IngestKeyRepository,
	auth *AuthHandler,
	core *CoreHandler,
	ingest *IngestHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Public runtime config for the SPA (e.g. the externally reachable ingest URL).
	r.GET("/api/v1/meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ingest_url":  cfg.PublicIngestURL,
			"agent_image": cfg.AgentImage,
			"self_update": core.SelfUpdateEnabled(),
		})
	})

	// --- Telemetry ingestion (ingest-key auth, not JWT) ---
	// OTLP/HTTP: the OTel Collector's otlphttp exporter posts to /otlp/v1/*.
	ingestGroup := r.Group("")
	ingestGroup.Use(middleware.IngestAuth(keys))
	{
		ingestGroup.POST("/otlp/v1/metrics", ingest.OTLPMetrics)
		ingestGroup.POST("/otlp/v1/logs", ingest.OTLPLogs)
		ingestGroup.POST("/otlp/v1/traces", ingest.OTLPTraces)
		ingestGroup.POST("/api/v1/prom/write", ingest.PromRemoteWrite)
		ingestGroup.POST("/api/v1/ingest/metrics", ingest.IngestMetricsJSON)
		// agent control channel (agent dials out; key-authed)
		ingestGroup.GET("/api/v1/agent/connect", core.AgentConnect)
	}

	limiter := middleware.NewRateLimiter(50, 100) // 50 rps, burst 100 per IP

	api := r.Group("/api/v1")
	api.Use(limiter.Middleware())

	// --- Public auth routes ---
	a := api.Group("/auth")
	{
		a.POST("/register", auth.Register)
		a.POST("/login", auth.Login)
		a.POST("/refresh", auth.Refresh)
		a.POST("/logout", auth.Logout)
	}

	// --- Authenticated routes ---
	authed := api.Group("")
	authed.Use(middleware.Auth(tokens))
	{
		authed.GET("/auth/me", auth.Me)
		authed.POST("/self-update", core.SelfUpdate)

		authed.GET("/organizations", core.ListOrgs)
		authed.POST("/organizations", core.CreateOrg)
		authed.GET("/organizations/:orgId/members", core.ListMembers)
		authed.GET("/organizations/:orgId/projects", core.ListProjects)
		authed.POST("/organizations/:orgId/projects", core.CreateProject)

		p := authed.Group("/projects/:projectId")
		{
			p.GET("", core.GetProject)
			p.PUT("", core.UpdateProject)
			p.DELETE("", core.DeleteProject)

			p.GET("/dashboard", core.Dashboard)
			p.GET("/timeline", core.Timeline)
			p.GET("/analyze", core.Analyze)
			p.GET("/metrics", core.Metrics)
			p.GET("/logs", core.Logs)
			p.GET("/traces", core.ListTraces)
			p.GET("/traces/:traceId", core.GetTrace)

			p.GET("/services", core.ListServices)
			p.POST("/services", core.CreateService)

			p.GET("/deployments", core.ListDeployments)
			p.POST("/deployments", core.CreateDeployment)

			p.GET("/alerts", core.ListAlerts)
			p.POST("/alerts", core.CreateAlert)

			// notification channels (reusable alert targets)
			p.GET("/channels", core.ListChannels)
			p.POST("/channels", core.CreateChannel)
			p.DELETE("/channels/:channelId", core.DeleteChannel)
			p.POST("/channels/:channelId/test", core.TestChannel)

			// alert rules (alerting engine)
			p.GET("/alert-rules", core.ListAlertRules)
			p.POST("/alert-rules", core.CreateAlertRule)
			p.PUT("/alert-rules/:ruleId", core.UpdateAlertRule)
			p.DELETE("/alert-rules/:ruleId", core.DeleteAlertRule)

			// managed servers (remote agent management via Tailscale SSH)
			p.GET("/servers", core.ListServers)
			p.POST("/servers", core.AddServer)
			p.DELETE("/servers/:serverId", core.DeleteServer)
			p.POST("/servers/:serverId/install", core.InstallAgent)
			p.POST("/servers/:serverId/beyla", core.InstallBeyla)
			p.POST("/servers/:serverId/beyla/remove", core.RemoveBeyla)
			p.POST("/servers/:serverId/remove", core.RemoveAgent)
			p.POST("/servers/:serverId/status", core.ServerStatus)
			p.POST("/servers/:serverId/run", core.RunServerCommand)
			p.GET("/audit", core.ListAudit)

			// agent control channel (live agents + commands)
			p.GET("/agents", core.ListAgents)
			p.POST("/agents/:agentId/command", core.SendCommand)

			// ingest key management (server onboarding)
			p.GET("/ingest-keys", core.ListIngestKeys)
			p.POST("/ingest-keys", core.CreateIngestKey)
			p.DELETE("/ingest-keys/:keyId", core.DeleteIngestKey)

			// ingestion
			p.POST("/ingest/metrics", core.IngestMetrics)
			p.POST("/ingest/logs", core.IngestLogs)

			// realtime
			p.GET("/ws", core.WS)
		}

		authed.PUT("/services/:serviceId", core.UpdateService)
		authed.DELETE("/services/:serviceId", core.DeleteService)
		authed.POST("/alerts/:alertId/resolve", core.ResolveAlert)
	}

	return r
}
