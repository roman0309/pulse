package repositories

import (
	"context"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
)

// UserRepository persists users and refresh tokens.
type UserRepository interface {
	CreateUser(ctx context.Context, u *entities.User) error
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)

	SaveRefreshToken(ctx context.Context, t *entities.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*entities.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
}

// OrganizationRepository manages organizations and membership.
type OrganizationRepository interface {
	Create(ctx context.Context, org *entities.Organization) error
	ListForUser(ctx context.Context, userID uuid.UUID) ([]entities.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Organization, error)
	AddMember(ctx context.Context, m *entities.TeamMember) error
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]entities.TeamMember, error)
	GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (entities.OrgRole, error)
}

// ProjectRepository manages projects.
type ProjectRepository interface {
	Create(ctx context.Context, p *entities.Project) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]entities.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error)
	Update(ctx context.Context, p *entities.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ServiceRepository manages services.
type ServiceRepository interface {
	Create(ctx context.Context, s *entities.Service) error
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.Service, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Service, error)
	Update(ctx context.Context, s *entities.Service) error
	Delete(ctx context.Context, id uuid.UUID) error
	// GetOrCreateByName resolves a service by (project, name, env), creating it
	// on first sight. Used by the ingestion pipeline for unknown services.
	GetOrCreateByName(ctx context.Context, projectID uuid.UUID, name, env string) (uuid.UUID, error)
}

// ServerRepository manages remote servers Pulse can manage over Tailscale SSH.
type ServerRepository interface {
	Create(ctx context.Context, s *entities.ManagedServer) error
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.ManagedServer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ManagedServer, error)
	Update(ctx context.Context, s *entities.ManagedServer) error
	Delete(ctx context.Context, projectID, id uuid.UUID) error
}

// IngestKeyRepository manages ingestion API keys.
type IngestKeyRepository interface {
	ResolveProject(ctx context.Context, keyHash string) (uuid.UUID, error)
	Create(ctx context.Context, k *entities.IngestKey) error
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.IngestKey, error)
	Delete(ctx context.Context, projectID, keyID uuid.UUID) error
}

// DeploymentRepository manages deployments.
type DeploymentRepository interface {
	Create(ctx context.Context, d *entities.Deployment) error
	ListByProject(ctx context.Context, projectID uuid.UUID, serviceID *uuid.UUID, limit int) ([]entities.Deployment, error)
	CountToday(ctx context.Context, projectID uuid.UUID) (int, error)
}

// AlertRepository manages alerts.
type AlertRepository interface {
	Create(ctx context.Context, a *entities.Alert) error
	ListByProject(ctx context.Context, projectID uuid.UUID, status *entities.AlertStatus) ([]entities.Alert, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Alert, error)
	Resolve(ctx context.Context, id uuid.UUID, resolvedAt time.Time) error
	CountActive(ctx context.Context, projectID uuid.UUID) (int, error)
}

// TimelineRepository manages the unified timeline event store.
type TimelineRepository interface {
	Create(ctx context.Context, e *entities.TimelineEvent) error
	ListByProject(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]entities.TimelineEvent, error)
}

// MetricRepository (ClickHouse) stores and queries time-series metrics.
type MetricRepository interface {
	Insert(ctx context.Context, points []entities.MetricPoint) error
	Query(ctx context.Context, projectID string, serviceID, metricName string, from, to time.Time, stepSeconds int) ([]entities.MetricSeries, error)
	Latest(ctx context.Context, projectID, serviceID, metricName string) (float64, error)
	// EvalValue returns the average value of a metric over [since, now] for the
	// alerting engine. ok is false when there are no data points in the window.
	EvalValue(ctx context.Context, projectID, serviceID, metricName string, since time.Time) (value float64, ok bool, err error)
	// DeleteService purges all metrics for one service.
	DeleteService(ctx context.Context, projectID, serviceID string) error
}

// AlertRuleRepository manages alert rule definitions and their evaluator state.
type AlertRuleRepository interface {
	Create(ctx context.Context, r *entities.AlertRule) error
	Update(ctx context.Context, r *entities.AlertRule) error
	Delete(ctx context.Context, projectID, ruleID uuid.UUID) error
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.AlertRule, error)
	ListEnabled(ctx context.Context) ([]entities.AlertRule, error)
	SetState(ctx context.Context, ruleID uuid.UUID, breachingSince *time.Time, activeAlertID *uuid.UUID) error
}

// LogRepository (ClickHouse) stores and queries structured logs.
type LogRepository interface {
	Insert(ctx context.Context, logs []entities.LogEntry) error
	Query(ctx context.Context, projectID string, serviceID, level, search string, from, to time.Time, limit, offset int) ([]entities.LogEntry, error)
	// DeleteService purges all logs for one service.
	DeleteService(ctx context.Context, projectID, serviceID string) error
}
