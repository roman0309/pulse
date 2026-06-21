package entities

import (
	"time"

	"github.com/google/uuid"
)

// ---------- Auth / Identity ----------

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type IngestKey struct {
	ID        uuid.UUID  `json:"id"`
	ProjectID uuid.UUID  `json:"project_id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	KeyHash   string     `json:"-"`
	Token     string     `json:"token,omitempty"` // plaintext, returned only once at creation
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------- Organizations ----------

type OrgRole string

const (
	RoleOwner  OrgRole = "owner"
	RoleAdmin  OrgRole = "admin"
	RoleMember OrgRole = "member"
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedBy uuid.UUID `json:"created_by"`
	Role      OrgRole   `json:"role,omitempty"` // role of the requesting user, when listed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamMember struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email,omitempty"`
	Name           string    `json:"name,omitempty"`
	Role           OrgRole   `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

// ---------- Managed servers (remote agent management via Tailscale SSH) ----------

type ManagedServer struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Name        string     `json:"name"`
	SSHTarget   string     `json:"ssh_target"` // user@host, for display
	SSHHost     string     `json:"ssh_host"`
	SSHPort     int        `json:"ssh_port"`
	SSHUser     string     `json:"ssh_user"`
	AuthMethod  string     `json:"auth_method"` // password | key
	SecretEnc   string     `json:"-"`           // encrypted credential, never serialized
	HostKey     string     `json:"host_key"`    // pinned SSH host-key fingerprint (TOFU)
	Status      string     `json:"status"`
	LastResult  string     `json:"last_result"`
	IngestKeyID *uuid.UUID `json:"-"`
	BeylaKeyID  *uuid.UUID `json:"-"` // ingest key for the zero-code app agent (Beyla)
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AuditEntry struct {
	ID        uuid.UUID  `json:"id"`
	ProjectID uuid.UUID  `json:"project_id"`
	UserID    *uuid.UUID `json:"user_id"`
	ServerID  *uuid.UUID `json:"server_id"`
	Action    string     `json:"action"`
	Detail    string     `json:"detail"`
	Success   bool       `json:"success"`
	CreatedAt time.Time  `json:"created_at"`
}

// Span is a single operation within a distributed trace.
type Span struct {
	ProjectID   string    `json:"project_id"`
	TraceID     string    `json:"trace_id"`
	SpanID      string    `json:"span_id"`
	ParentID    string    `json:"parent_id"`
	ServiceName string    `json:"service_name"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	StatusCode  string    `json:"status_code"` // unset | ok | error
	StartTime   time.Time `json:"start_time"`
	DurationMS  float64   `json:"duration_ms"`
	Attributes  string    `json:"attributes"`
}

// TraceSummary is an aggregate row for the trace list (one per trace_id).
type TraceSummary struct {
	TraceID     string    `json:"trace_id"`
	RootService string    `json:"root_service"`
	RootName    string    `json:"root_name"`
	StartTime   time.Time `json:"start_time"`
	DurationMS  float64   `json:"duration_ms"`
	SpanCount   int       `json:"span_count"`
	ErrorCount  int       `json:"error_count"`
}

// ---------- Projects ----------

type Project struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ---------- Services ----------

type ServiceStatus string

const (
	StatusHealthy  ServiceStatus = "healthy"
	StatusDegraded ServiceStatus = "degraded"
	StatusDown     ServiceStatus = "down"
)

type Service struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Environment string        `json:"environment"`
	Status      ServiceStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ---------- Deployments ----------

type Deployment struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	ServiceID   uuid.UUID `json:"service_id"`
	ServiceName string    `json:"service_name,omitempty"`
	Version     string    `json:"version"`
	CommitSHA   string    `json:"commit_sha"`
	Environment string    `json:"environment"`
	DeployedBy  string    `json:"deployed_by"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ---------- Alerts ----------

type AlertSeverity string
type AlertStatus string
type AlertType string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"

	AlertActive   AlertStatus = "active"
	AlertResolved AlertStatus = "resolved"

	TypeHighLatency   AlertType = "high_latency"
	TypeHighErrorRate AlertType = "high_error_rate"
	TypeServiceDown   AlertType = "service_down"
)

type Alert struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	ServiceID   *uuid.UUID    `json:"service_id"`
	ServiceName string        `json:"service_name,omitempty"`
	Title       string        `json:"title"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Status      AlertStatus   `json:"status"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	ResolvedAt  *time.Time    `json:"resolved_at"`
}

// ---------- Timeline ----------

type TimelineEventType string

const (
	EventDeployment  TimelineEventType = "deployment"
	EventAlert       TimelineEventType = "alert"
	EventMetricSpike TimelineEventType = "metric_spike"
	EventErrorSpike  TimelineEventType = "error_spike"
	EventRecovery    TimelineEventType = "recovery"
)

type TimelineEvent struct {
	ID          uuid.UUID         `json:"id"`
	ProjectID   uuid.UUID         `json:"project_id"`
	ServiceID   *uuid.UUID        `json:"service_id"`
	ServiceName string            `json:"service_name,omitempty"`
	Type        TimelineEventType `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    *AlertSeverity    `json:"severity"`
	RefID       *uuid.UUID        `json:"ref_id"`
	OccurredAt  time.Time         `json:"occurred_at"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ---------- Alert rules ----------

type RuleOperator string

const (
	OpGreater      RuleOperator = "gt"
	OpLess         RuleOperator = "lt"
	OpGreaterEqual RuleOperator = "gte"
	OpLessEqual    RuleOperator = "lte"
)

type AlertRule struct {
	ID         uuid.UUID     `json:"id"`
	ProjectID  uuid.UUID     `json:"project_id"`
	Name       string        `json:"name"`
	ServiceID  *uuid.UUID    `json:"service_id"`
	Metric     string        `json:"metric"`
	Operator   RuleOperator  `json:"operator"`
	Threshold  float64       `json:"threshold"`
	ForSeconds int           `json:"for_seconds"`
	Severity   AlertSeverity `json:"severity"`
	Type       AlertType     `json:"type"`
	NotifyType      string     `json:"notify_type"` // none | slack | telegram | webhook (legacy inline)
	NotifyURL       string     `json:"notify_url"`
	NotifyChannelID *uuid.UUID `json:"notify_channel_id"` // preferred: a saved channel
	Enabled         bool       `json:"enabled"`

	BreachingSince *time.Time `json:"-"`
	ActiveAlertID  *uuid.UUID `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationChannel is a reusable, project-scoped delivery target. The
// secret (bot token / webhook URL) is stored encrypted in ConfigEnc and never
// serialized; the UI gets a non-secret Hint instead.
type NotificationChannel struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // slack | telegram | webhook
	ConfigEnc string    `json:"-"`
	Hint      string    `json:"hint"` // computed, e.g. "chat 12345" or "hooks.slack.com"
	CreatedAt time.Time `json:"created_at"`
}

// Breached reports whether value violates the rule's condition.
func (r *AlertRule) Breached(value float64) bool {
	switch r.Operator {
	case OpGreater:
		return value > r.Threshold
	case OpLess:
		return value < r.Threshold
	case OpGreaterEqual:
		return value >= r.Threshold
	case OpLessEqual:
		return value <= r.Threshold
	}
	return false
}

// ---------- Metrics (ClickHouse) ----------

type MetricPoint struct {
	ProjectID   string    `json:"project_id"`
	ServiceID   string    `json:"service_id"`
	ServiceName string    `json:"service_name"`
	MetricName  string    `json:"metric_name"`
	Value       float64   `json:"value"`
	Timestamp   time.Time `json:"timestamp"`
}

// MetricSeries is an aggregated series for charting.
type MetricSeries struct {
	MetricName string        `json:"metric_name"`
	ServiceID  string        `json:"service_id"`
	Points     []SeriesPoint `json:"points"`
}

type SeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ---------- Logs (ClickHouse) ----------

type LogEntry struct {
	ProjectID   string    `json:"project_id"`
	ServiceID   string    `json:"service_id"`
	ServiceName string    `json:"service_name"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Metadata    string    `json:"metadata"`
	TraceID     string    `json:"trace_id"`
	Timestamp   time.Time `json:"timestamp"`
}
