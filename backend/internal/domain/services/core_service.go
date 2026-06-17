package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/acme/observability/internal/analyzer"
	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/repositories"
	"github.com/acme/observability/internal/remote"
	"github.com/acme/observability/internal/ws"
	"github.com/acme/observability/pkg/hash"
	"github.com/google/uuid"
)

var ErrForbidden = errors.New("forbidden")

// CoreService is the application service layer for the observability domain.
// It coordinates repositories, the realtime hub and the RCA analyzer.
type CoreService struct {
	Orgs        repositories.OrganizationRepository
	Projects    repositories.ProjectRepository
	Services    repositories.ServiceRepository
	Deployments repositories.DeploymentRepository
	Alerts      repositories.AlertRepository
	Timeline    repositories.TimelineRepository
	Metrics     repositories.MetricRepository
	Logs        repositories.LogRepository
	IngestKeys  repositories.IngestKeyRepository
	AlertRules  repositories.AlertRuleRepository
	Servers     repositories.ServerRepository
	Analyzer    analyzer.Analyzer
	Hub         *ws.Hub
	Exec        remote.Executor
	// PublicIngestURL is the address agents push to; injected into remote installs.
	PublicIngestURL string
}

const agentInstallerURL = "https://raw.githubusercontent.com/roman0309/pulse/main/deploy/install-agent.sh"

// ---------- Managed servers (remote agent management over Tailscale SSH) ----------

func (s *CoreService) ListServers(ctx context.Context, userID, projectID uuid.UUID) ([]entities.ManagedServer, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Servers.ListByProject(ctx, projectID)
}

func (s *CoreService) AddServer(ctx context.Context, userID, projectID uuid.UUID, name, sshTarget string) (*entities.ManagedServer, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	if !remote.ValidateTarget(sshTarget) {
		return nil, remote.ErrBadTarget
	}
	if name == "" {
		name = sshTarget
	}
	srv := &entities.ManagedServer{ProjectID: projectID, Name: name, SSHTarget: sshTarget, Status: "pending"}
	return srv, s.Servers.Create(ctx, srv)
}

func (s *CoreService) DeleteServer(ctx context.Context, userID, projectID, serverID uuid.UUID) error {
	srv, err := s.Servers.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if _, err := s.requireProjectAccess(ctx, userID, srv.ProjectID); err != nil {
		return err
	}
	if srv.IngestKeyID != nil {
		_ = s.IngestKeys.Delete(ctx, projectID, *srv.IngestKeyID)
	}
	return s.Servers.Delete(ctx, projectID, serverID)
}

// InstallAgent provisions the host agent on the server via Tailscale SSH. It
// mints a dedicated ingest key for the server and passes it to the installer.
func (s *CoreService) InstallAgent(ctx context.Context, userID, serverID uuid.UUID) (*entities.ManagedServer, error) {
	srv, err := s.serverForUser(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	key, err := s.CreateIngestKey(ctx, userID, srv.ProjectID, srv.Name)
	if err != nil {
		return nil, err
	}
	endpoint := s.PublicIngestURL
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	cmd := fmt.Sprintf("curl -fsSL %s | PULSE_ENDPOINT=%s PULSE_KEY=%s PULSE_SERVICE=%s sh",
		agentInstallerURL, endpoint, key.Token, srv.Name)
	out, runErr := s.Exec.Run(ctx, srv.SSHTarget, cmd)

	srv.IngestKeyID = &key.ID
	srv.LastResult = truncate(out, 2000)
	if runErr != nil {
		srv.Status = "error"
		srv.LastResult = truncate(out+"\n"+runErr.Error(), 2000)
	} else {
		srv.Status = "installed"
	}
	_ = s.Servers.Update(ctx, srv)
	return srv, nil
}

// RemoveAgent stops the agent on the server and revokes its ingest key.
func (s *CoreService) RemoveAgent(ctx context.Context, userID, serverID uuid.UUID) (*entities.ManagedServer, error) {
	srv, err := s.serverForUser(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	out, runErr := s.Exec.Run(ctx, srv.SSHTarget, "docker rm -f pulse-agent pulse-beyla 2>/dev/null; echo removed")
	if srv.IngestKeyID != nil {
		_ = s.IngestKeys.Delete(ctx, srv.ProjectID, *srv.IngestKeyID)
		srv.IngestKeyID = nil
	}
	srv.LastResult = truncate(out, 2000)
	srv.Status = "removed"
	if runErr != nil {
		srv.Status = "error"
		srv.LastResult = truncate(out+"\n"+runErr.Error(), 2000)
	}
	_ = s.Servers.Update(ctx, srv)
	return srv, nil
}

// CheckStatus queries whether the agent container is running on the server.
func (s *CoreService) CheckStatus(ctx context.Context, userID, serverID uuid.UUID) (*entities.ManagedServer, error) {
	srv, err := s.serverForUser(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	out, runErr := s.Exec.Run(ctx, srv.SSHTarget,
		"docker ps --filter name=pulse-agent --filter name=pulse-beyla --format '{{.Names}} {{.Status}}'")
	srv.LastResult = truncate(out, 2000)
	if runErr != nil {
		srv.LastResult = truncate(out+"\n"+runErr.Error(), 2000)
	}
	_ = s.Servers.Update(ctx, srv)
	return srv, nil
}

func (s *CoreService) serverForUser(ctx context.Context, userID, serverID uuid.UUID) (*entities.ManagedServer, error) {
	srv, err := s.Servers.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectAccess(ctx, userID, srv.ProjectID); err != nil {
		return nil, err
	}
	if s.Exec == nil {
		return nil, errors.New("remote execution is not enabled on this Pulse instance")
	}
	return srv, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// ---------- Organizations ----------

func (s *CoreService) CreateOrg(ctx context.Context, userID uuid.UUID, name string) (*entities.Organization, error) {
	org := &entities.Organization{Name: name, Slug: slugify(name), CreatedBy: userID}
	if err := s.Orgs.Create(ctx, org); err != nil {
		return nil, err
	}
	if err := s.Orgs.AddMember(ctx, &entities.TeamMember{
		OrganizationID: org.ID, UserID: userID, Role: entities.RoleOwner,
	}); err != nil {
		return nil, err
	}
	org.Role = entities.RoleOwner
	return org, nil
}

func (s *CoreService) ListOrgs(ctx context.Context, userID uuid.UUID) ([]entities.Organization, error) {
	return s.Orgs.ListForUser(ctx, userID)
}

func (s *CoreService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]entities.TeamMember, error) {
	return s.Orgs.ListMembers(ctx, orgID)
}

// requireMember verifies the user belongs to the org owning the project.
func (s *CoreService) requireProjectAccess(ctx context.Context, userID, projectID uuid.UUID) (*entities.Project, error) {
	p, err := s.Projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Orgs.GetMemberRole(ctx, p.OrganizationID, userID); err != nil {
		return nil, ErrForbidden
	}
	return p, nil
}

// ---------- Projects ----------

func (s *CoreService) CreateProject(ctx context.Context, userID, orgID uuid.UUID, name, description string) (*entities.Project, error) {
	if _, err := s.Orgs.GetMemberRole(ctx, orgID, userID); err != nil {
		return nil, ErrForbidden
	}
	p := &entities.Project{OrganizationID: orgID, Name: name, Slug: slugify(name), Description: description}
	return p, s.Projects.Create(ctx, p)
}

func (s *CoreService) ListProjects(ctx context.Context, userID, orgID uuid.UUID) ([]entities.Project, error) {
	if _, err := s.Orgs.GetMemberRole(ctx, orgID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.Projects.ListByOrg(ctx, orgID)
}

func (s *CoreService) GetProject(ctx context.Context, userID, projectID uuid.UUID) (*entities.Project, error) {
	return s.requireProjectAccess(ctx, userID, projectID)
}

func (s *CoreService) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, name, description string) (*entities.Project, error) {
	p, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	p.Name, p.Description = name, description
	return p, s.Projects.Update(ctx, p)
}

func (s *CoreService) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return err
	}
	return s.Projects.Delete(ctx, projectID)
}

// ---------- Services ----------

func (s *CoreService) CreateService(ctx context.Context, userID, projectID uuid.UUID, name, env string, status entities.ServiceStatus) (*entities.Service, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	if status == "" {
		status = entities.StatusHealthy
	}
	if env == "" {
		env = "production"
	}
	svc := &entities.Service{ProjectID: projectID, Name: name, Environment: env, Status: status}
	return svc, s.Services.Create(ctx, svc)
}

func (s *CoreService) ListServices(ctx context.Context, userID, projectID uuid.UUID) ([]entities.Service, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Services.ListByProject(ctx, projectID)
}

func (s *CoreService) UpdateService(ctx context.Context, userID, serviceID uuid.UUID, name, env string, status entities.ServiceStatus) (*entities.Service, error) {
	svc, err := s.Services.GetByID(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectAccess(ctx, userID, svc.ProjectID); err != nil {
		return nil, err
	}
	svc.Name, svc.Environment, svc.Status = name, env, status
	return svc, s.Services.Update(ctx, svc)
}

func (s *CoreService) DeleteService(ctx context.Context, userID, serviceID uuid.UUID) error {
	svc, err := s.Services.GetByID(ctx, serviceID)
	if err != nil {
		return err
	}
	if _, err := s.requireProjectAccess(ctx, userID, svc.ProjectID); err != nil {
		return err
	}
	if err := s.Services.Delete(ctx, serviceID); err != nil {
		return err
	}
	// Purge the service's time-series data too (best-effort).
	pid := svc.ProjectID.String()
	_ = s.Metrics.DeleteService(ctx, pid, serviceID.String())
	_ = s.Logs.DeleteService(ctx, pid, serviceID.String())
	return nil
}

// ---------- Deployments ----------

func (s *CoreService) CreateDeployment(ctx context.Context, userID uuid.UUID, d *entities.Deployment) error {
	if _, err := s.requireProjectAccess(ctx, userID, d.ProjectID); err != nil {
		return err
	}
	if d.Status == "" {
		d.Status = "success"
	}
	if err := s.Deployments.Create(ctx, d); err != nil {
		return err
	}
	// Auto-create a timeline event for the deployment.
	svcID := d.ServiceID
	ev := &entities.TimelineEvent{
		ProjectID:   d.ProjectID,
		ServiceID:   &svcID,
		Type:        entities.EventDeployment,
		Title:       "Deployment " + d.Version,
		Description: d.ServiceName + " deployed by " + d.DeployedBy,
		OccurredAt:  d.CreatedAt,
	}
	_ = s.Timeline.Create(ctx, ev)
	s.broadcast(d.ProjectID, "timeline", ev)
	return nil
}

func (s *CoreService) ListDeployments(ctx context.Context, userID, projectID uuid.UUID, serviceID *uuid.UUID, limit int) ([]entities.Deployment, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Deployments.ListByProject(ctx, projectID, serviceID, limit)
}

// ---------- Alerts ----------

func (s *CoreService) CreateAlert(ctx context.Context, userID uuid.UUID, a *entities.Alert) error {
	if _, err := s.requireProjectAccess(ctx, userID, a.ProjectID); err != nil {
		return err
	}
	if a.Status == "" {
		a.Status = entities.AlertActive
	}
	if err := s.Alerts.Create(ctx, a); err != nil {
		return err
	}
	sev := a.Severity
	ev := &entities.TimelineEvent{
		ProjectID:   a.ProjectID,
		ServiceID:   a.ServiceID,
		Type:        entities.EventAlert,
		Title:       "Alert triggered",
		Description: a.Title,
		Severity:    &sev,
		RefID:       &a.ID,
		OccurredAt:  a.CreatedAt,
	}
	_ = s.Timeline.Create(ctx, ev)
	s.broadcast(a.ProjectID, "alert", a)
	s.broadcast(a.ProjectID, "timeline", ev)
	return nil
}

func (s *CoreService) ListAlerts(ctx context.Context, userID, projectID uuid.UUID, status *entities.AlertStatus) ([]entities.Alert, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Alerts.ListByProject(ctx, projectID, status)
}

func (s *CoreService) ResolveAlert(ctx context.Context, userID, alertID uuid.UUID) (*entities.Alert, error) {
	a, err := s.Alerts.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectAccess(ctx, userID, a.ProjectID); err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.Alerts.Resolve(ctx, alertID, now); err != nil {
		return nil, err
	}
	a.Status = entities.AlertResolved
	a.ResolvedAt = &now
	s.broadcast(a.ProjectID, "alert", a)
	return a, nil
}

// ---------- Timeline ----------

func (s *CoreService) Timeline_(ctx context.Context, userID, projectID uuid.UUID, from, to time.Time) ([]entities.TimelineEvent, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Timeline.ListByProject(ctx, projectID, from, to)
}

// ---------- Metrics ----------

func (s *CoreService) QueryMetrics(ctx context.Context, userID, projectID uuid.UUID, serviceID, metricName string, from, to time.Time, step int) ([]entities.MetricSeries, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Metrics.Query(ctx, projectID.String(), serviceID, metricName, from, to, step)
}

// ---------- Logs ----------

func (s *CoreService) QueryLogs(ctx context.Context, userID, projectID uuid.UUID, serviceID, level, search string, from, to time.Time, limit, offset int) ([]entities.LogEntry, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.Logs.Query(ctx, projectID.String(), serviceID, level, search, from, to, limit, offset)
}

// ---------- Root Cause Analysis ----------

func (s *CoreService) Analyze(ctx context.Context, userID, projectID uuid.UUID, window time.Duration) (*analyzer.Result, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	now := time.Now()
	from := now.Add(-window)
	pid := projectID.String()

	latency, _ := s.Metrics.Query(ctx, pid, "", "latency_p95", from, now, 60)
	errs, _ := s.Metrics.Query(ctx, pid, "", "error_rate", from, now, 60)
	logs, _ := s.Logs.Query(ctx, pid, "", "error", "", from, now, 500, 0)
	alerts, _ := s.Alerts.ListByProject(ctx, projectID, nil)
	deps, _ := s.Deployments.ListByProject(ctx, projectID, nil, 50)
	svcs, _ := s.Services.ListByProject(ctx, projectID)

	in := analyzer.Input{
		ProjectID:   pid,
		Window:      window,
		Now:         now,
		Metrics:     append(latency, errs...),
		Logs:        logs,
		Alerts:      alerts,
		Deployments: deps,
		Services:    svcs,
	}
	return s.Analyzer.Analyze(ctx, in)
}

// ---------- Dashboard ----------

type DashboardSummary struct {
	SystemHealth     string  `json:"system_health"` // healthy | degraded | down
	ActiveAlerts     int     `json:"active_alerts"`
	DeploymentsToday int     `json:"deployments_today"`
	ErrorRate        float64 `json:"error_rate"`
}

func (s *CoreService) Dashboard(ctx context.Context, userID, projectID uuid.UUID) (*DashboardSummary, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	active, _ := s.Alerts.CountActive(ctx, projectID)
	deploys, _ := s.Deployments.CountToday(ctx, projectID)
	errRate, _ := s.Metrics.Latest(ctx, projectID.String(), "", "error_rate")

	svcs, _ := s.Services.ListByProject(ctx, projectID)
	health := "healthy"
	for _, sv := range svcs {
		if sv.Status == entities.StatusDown {
			health = "down"
			break
		}
		if sv.Status == entities.StatusDegraded {
			health = "degraded"
		}
	}

	return &DashboardSummary{
		SystemHealth:     health,
		ActiveAlerts:     active,
		DeploymentsToday: deploys,
		ErrorRate:        errRate,
	}, nil
}

// ---------- Ingestion (metrics/logs) ----------

func (s *CoreService) IngestMetrics(ctx context.Context, points []entities.MetricPoint) error {
	if err := s.Metrics.Insert(ctx, points); err != nil {
		return err
	}
	for _, p := range points {
		s.broadcast(uuid.MustParse(p.ProjectID), "metric", p)
	}
	return nil
}

func (s *CoreService) IngestLogs(ctx context.Context, logs []entities.LogEntry) error {
	return s.Logs.Insert(ctx, logs)
}

// ---------- Alert rules ----------

func (s *CoreService) ListAlertRules(ctx context.Context, userID, projectID uuid.UUID) ([]entities.AlertRule, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.AlertRules.ListByProject(ctx, projectID)
}

func (s *CoreService) CreateAlertRule(ctx context.Context, userID uuid.UUID, rule *entities.AlertRule) error {
	if _, err := s.requireProjectAccess(ctx, userID, rule.ProjectID); err != nil {
		return err
	}
	return s.AlertRules.Create(ctx, rule)
}

func (s *CoreService) UpdateAlertRule(ctx context.Context, userID uuid.UUID, rule *entities.AlertRule) error {
	if _, err := s.requireProjectAccess(ctx, userID, rule.ProjectID); err != nil {
		return err
	}
	return s.AlertRules.Update(ctx, rule)
}

func (s *CoreService) DeleteAlertRule(ctx context.Context, userID, projectID, ruleID uuid.UUID) error {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return err
	}
	return s.AlertRules.Delete(ctx, projectID, ruleID)
}

// ---------- Ingest keys (server onboarding) ----------

func (s *CoreService) CreateIngestKey(ctx context.Context, userID, projectID uuid.UUID, name string) (*entities.IngestKey, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	if name == "" {
		name = "server"
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	k := &entities.IngestKey{
		ProjectID: projectID,
		Name:      name,
		Prefix:    token[:12],
		KeyHash:   hash.SHA256(token),
	}
	if err := s.IngestKeys.Create(ctx, k); err != nil {
		return nil, err
	}
	k.Token = token // returned once, never stored in plaintext
	return k, nil
}

func (s *CoreService) ListIngestKeys(ctx context.Context, userID, projectID uuid.UUID) ([]entities.IngestKey, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.IngestKeys.ListByProject(ctx, projectID)
}

func (s *CoreService) DeleteIngestKey(ctx context.Context, userID, projectID, keyID uuid.UUID) error {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return err
	}
	return s.IngestKeys.Delete(ctx, projectID, keyID)
}

// generateToken returns a random ingest token like "pls_<48 hex chars>".
func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "pls_" + hex.EncodeToString(b), nil
}

// ResolveService resolves (and auto-creates) a service by name within a project.
// Used by the OTLP / remote_write ingestion pipeline for services that were not
// registered manually.
func (s *CoreService) ResolveService(ctx context.Context, projectID uuid.UUID, name, env string) (uuid.UUID, error) {
	return s.Services.GetOrCreateByName(ctx, projectID, name, env)
}

func (s *CoreService) broadcast(projectID uuid.UUID, kind string, payload interface{}) {
	if s.Hub == nil {
		return
	}
	s.Hub.Broadcast(ws.Event{Type: kind, ProjectID: projectID.String(), Payload: payload})
}
