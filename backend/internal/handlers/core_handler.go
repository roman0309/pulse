package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/middleware"
	"github.com/acme/observability/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CoreHandler struct {
	core *services.CoreService
	hub  *ws.Hub
}

func NewCoreHandler(core *services.CoreService, hub *ws.Hub) *CoreHandler {
	return &CoreHandler{core: core, hub: hub}
}

// ---------- Organizations ----------

func (h *CoreHandler) ListOrgs(c *gin.Context) {
	orgs, err := h.core.ListOrgs(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func (h *CoreHandler) CreateOrg(c *gin.Context) {
	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	org, err := h.core.CreateOrg(c.Request.Context(), middleware.UserID(c), req.Name)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, org)
}

func (h *CoreHandler) ListMembers(c *gin.Context) {
	orgID, ok := parseUUIDParam(c, "orgId")
	if !ok {
		return
	}
	members, err := h.core.ListMembers(c.Request.Context(), orgID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// ---------- Projects ----------

func (h *CoreHandler) ListProjects(c *gin.Context) {
	orgID, ok := parseUUIDParam(c, "orgId")
	if !ok {
		return
	}
	projects, err := h.core.ListProjects(c.Request.Context(), middleware.UserID(c), orgID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *CoreHandler) CreateProject(c *gin.Context) {
	orgID, ok := parseUUIDParam(c, "orgId")
	if !ok {
		return
	}
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	p, err := h.core.CreateProject(c.Request.Context(), middleware.UserID(c), orgID, req.Name, req.Description)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *CoreHandler) GetProject(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	p, err := h.core.GetProject(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *CoreHandler) UpdateProject(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	p, err := h.core.UpdateProject(c.Request.Context(), middleware.UserID(c), projectID, req.Name, req.Description)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *CoreHandler) DeleteProject(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	if err := h.core.DeleteProject(c.Request.Context(), middleware.UserID(c), projectID); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- Services ----------

func (h *CoreHandler) ListServices(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	svcs, err := h.core.ListServices(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"services": svcs})
}

func (h *CoreHandler) CreateService(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	svc, err := h.core.CreateService(c.Request.Context(), middleware.UserID(c), projectID,
		req.Name, req.Environment, entities.ServiceStatus(req.Status))
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, svc)
}

func (h *CoreHandler) UpdateService(c *gin.Context) {
	serviceID, ok := parseUUIDParam(c, "serviceId")
	if !ok {
		return
	}
	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	svc, err := h.core.UpdateService(c.Request.Context(), middleware.UserID(c), serviceID,
		req.Name, req.Environment, entities.ServiceStatus(req.Status))
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, svc)
}

func (h *CoreHandler) DeleteService(c *gin.Context) {
	serviceID, ok := parseUUIDParam(c, "serviceId")
	if !ok {
		return
	}
	if err := h.core.DeleteService(c.Request.Context(), middleware.UserID(c), serviceID); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- Deployments ----------

func (h *CoreHandler) ListDeployments(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var serviceID *uuid.UUID
	if v := c.Query("service_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			serviceID = &id
		}
	}
	deps, err := h.core.ListDeployments(c.Request.Context(), middleware.UserID(c), projectID, serviceID, queryInt(c, "limit", 100))
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deployments": deps})
}

func (h *CoreHandler) CreateDeployment(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	serviceID, _ := uuid.Parse(req.ServiceID)
	d := &entities.Deployment{
		ProjectID:   projectID,
		ServiceID:   serviceID,
		Version:     req.Version,
		CommitSHA:   req.CommitSHA,
		Environment: req.Environment,
		DeployedBy:  req.DeployedBy,
		Status:      req.Status,
		CreatedAt:   time.Now(),
	}
	if err := h.core.CreateDeployment(c.Request.Context(), middleware.UserID(c), d); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

// ---------- Alerts ----------

func (h *CoreHandler) ListAlerts(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var status *entities.AlertStatus
	if v := c.Query("status"); v == "active" || v == "resolved" {
		s := entities.AlertStatus(v)
		status = &s
	}
	alerts, err := h.core.ListAlerts(c.Request.Context(), middleware.UserID(c), projectID, status)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *CoreHandler) CreateAlert(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	a := &entities.Alert{
		ProjectID:   projectID,
		Title:       req.Title,
		Type:        entities.AlertType(req.Type),
		Severity:    entities.AlertSeverity(req.Severity),
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	if req.ServiceID != "" {
		if id, err := uuid.Parse(req.ServiceID); err == nil {
			a.ServiceID = &id
		}
	}
	if err := h.core.CreateAlert(c.Request.Context(), middleware.UserID(c), a); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *CoreHandler) ResolveAlert(c *gin.Context) {
	alertID, ok := parseUUIDParam(c, "alertId")
	if !ok {
		return
	}
	a, err := h.core.ResolveAlert(c.Request.Context(), middleware.UserID(c), alertID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

// ---------- Timeline ----------

func (h *CoreHandler) Timeline(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	from, to := parseTimeRange(c, 24*time.Hour)
	events, err := h.core.Timeline_(c.Request.Context(), middleware.UserID(c), projectID, from, to)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// ---------- Metrics ----------

func (h *CoreHandler) Metrics(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	metricName := c.Query("metric")
	if metricName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric query param is required"})
		return
	}
	from, to := parseTimeRange(c, time.Hour)
	step := queryInt(c, "step", 60)
	series, err := h.core.QueryMetrics(c.Request.Context(), middleware.UserID(c), projectID,
		c.Query("service_id"), metricName, from, to, step)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"series": series})
}

// ---------- Logs ----------

func (h *CoreHandler) Logs(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	from, to := parseTimeRange(c, 24*time.Hour)
	logs, err := h.core.QueryLogs(c.Request.Context(), middleware.UserID(c), projectID,
		c.Query("service_id"), c.Query("level"), c.Query("search"),
		from, to, queryInt(c, "limit", 100), queryInt(c, "offset", 0))
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// ---------- Dashboard ----------

func (h *CoreHandler) Dashboard(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	summary, err := h.core.Dashboard(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// ---------- Managed servers ----------

func (h *CoreHandler) ListServers(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	servers, err := h.core.ListServers(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

func (h *CoreHandler) AddServer(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req struct {
		Name      string `json:"name" binding:"max=100"`
		SSHTarget string `json:"ssh_target" binding:"required,max=200"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	srv, err := h.core.AddServer(c.Request.Context(), middleware.UserID(c), projectID, req.Name, req.SSHTarget)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, srv)
}

func (h *CoreHandler) DeleteServer(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	serverID, ok := parseUUIDParam(c, "serverId")
	if !ok {
		return
	}
	if err := h.core.DeleteServer(c.Request.Context(), middleware.UserID(c), projectID, serverID); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *CoreHandler) serverAction(c *gin.Context, fn func(ctx context.Context, userID, serverID uuid.UUID) (*entities.ManagedServer, error)) {
	serverID, ok := parseUUIDParam(c, "serverId")
	if !ok {
		return
	}
	srv, err := fn(c.Request.Context(), middleware.UserID(c), serverID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, srv)
}

func (h *CoreHandler) InstallAgent(c *gin.Context) { h.serverAction(c, h.core.InstallAgent) }
func (h *CoreHandler) RemoveAgent(c *gin.Context)  { h.serverAction(c, h.core.RemoveAgent) }
func (h *CoreHandler) ServerStatus(c *gin.Context) { h.serverAction(c, h.core.CheckStatus) }

// ---------- Alert rules ----------

func (h *CoreHandler) ListAlertRules(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	rules, err := h.core.ListAlertRules(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func ruleFromRequest(projectID uuid.UUID, req AlertRuleRequest) *entities.AlertRule {
	rule := &entities.AlertRule{
		ProjectID:  projectID,
		Name:       req.Name,
		Metric:     req.Metric,
		Operator:   entities.RuleOperator(req.Operator),
		Threshold:  req.Threshold,
		ForSeconds: req.ForSeconds,
		Severity:   entities.AlertSeverity(req.Severity),
		Type:       entities.AlertType(req.Type),
		NotifyType: req.NotifyType,
		NotifyURL:  req.NotifyURL,
		Enabled:    true,
	}
	if rule.NotifyType == "" {
		rule.NotifyType = "none"
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.ServiceID != "" {
		if id, err := uuid.Parse(req.ServiceID); err == nil {
			rule.ServiceID = &id
		}
	}
	return rule
}

func (h *CoreHandler) CreateAlertRule(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	rule := ruleFromRequest(projectID, req)
	if err := h.core.CreateAlertRule(c.Request.Context(), middleware.UserID(c), rule); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *CoreHandler) UpdateAlertRule(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	ruleID, ok := parseUUIDParam(c, "ruleId")
	if !ok {
		return
	}
	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	rule := ruleFromRequest(projectID, req)
	rule.ID = ruleID
	if err := h.core.UpdateAlertRule(c.Request.Context(), middleware.UserID(c), rule); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *CoreHandler) DeleteAlertRule(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	ruleID, ok := parseUUIDParam(c, "ruleId")
	if !ok {
		return
	}
	if err := h.core.DeleteAlertRule(c.Request.Context(), middleware.UserID(c), projectID, ruleID); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- Ingest keys (server onboarding) ----------

func (h *CoreHandler) ListIngestKeys(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	keys, err := h.core.ListIngestKeys(c.Request.Context(), middleware.UserID(c), projectID)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func (h *CoreHandler) CreateIngestKey(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name" binding:"max=100"`
	}
	_ = c.ShouldBindJSON(&req)
	key, err := h.core.CreateIngestKey(c.Request.Context(), middleware.UserID(c), projectID, req.Name)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, key)
}

func (h *CoreHandler) DeleteIngestKey(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	keyID, ok := parseUUIDParam(c, "keyId")
	if !ok {
		return
	}
	if err := h.core.DeleteIngestKey(c.Request.Context(), middleware.UserID(c), projectID, keyID); err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- Root Cause Analysis ----------

func (h *CoreHandler) Analyze(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	windowMin := queryInt(c, "window_minutes", 120)
	result, err := h.core.Analyze(c.Request.Context(), middleware.UserID(c), projectID, time.Duration(windowMin)*time.Minute)
	if err != nil {
		handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ---------- WebSocket ----------

func (h *CoreHandler) WS(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	// Access check via the regular service path.
	if _, err := h.core.GetProject(c.Request.Context(), middleware.UserID(c), projectID); err != nil {
		handleDomainError(c, err)
		return
	}
	h.hub.ServeWS(c, projectID.String())
}

// ---------- Ingestion ----------

func (h *CoreHandler) IngestMetrics(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req IngestMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	points := make([]entities.MetricPoint, 0, len(req.Points))
	for _, p := range req.Points {
		points = append(points, entities.MetricPoint{
			ProjectID:   projectID.String(),
			ServiceID:   p.ServiceID,
			ServiceName: p.ServiceName,
			MetricName:  p.MetricName,
			Value:       p.Value,
			Timestamp:   parseTimestamp(p.Timestamp),
		})
	}
	if err := h.core.IngestMetrics(c.Request.Context(), points); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(points)})
}

func (h *CoreHandler) IngestLogs(c *gin.Context) {
	projectID, ok := parseUUIDParam(c, "projectId")
	if !ok {
		return
	}
	var req IngestLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	logs := make([]entities.LogEntry, 0, len(req.Logs))
	for _, l := range req.Logs {
		logs = append(logs, entities.LogEntry{
			ProjectID:   projectID.String(),
			ServiceID:   l.ServiceID,
			ServiceName: l.ServiceName,
			Level:       l.Level,
			Message:     l.Message,
			Metadata:    l.Metadata,
			Timestamp:   parseTimestamp(l.Timestamp),
		})
	}
	if err := h.core.IngestLogs(c.Request.Context(), logs); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(logs)})
}
