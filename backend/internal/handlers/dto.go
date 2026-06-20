package handlers

// Request DTOs with validation tags (go-playground/validator via Gin binding).

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateOrgRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
}

type CreateServiceRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Environment string `json:"environment" binding:"max=50"`
	Status      string `json:"status" binding:"omitempty,oneof=healthy degraded down"`
}

type UpdateServiceRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Environment string `json:"environment" binding:"required,max=50"`
	Status      string `json:"status" binding:"required,oneof=healthy degraded down"`
}

type CreateDeploymentRequest struct {
	ServiceID   string `json:"service_id" binding:"required,uuid"`
	Version     string `json:"version" binding:"required,max=50"`
	CommitSHA   string `json:"commit_sha" binding:"max=64"`
	Environment string `json:"environment" binding:"max=50"`
	DeployedBy  string `json:"deployed_by" binding:"max=100"`
	Status      string `json:"status" binding:"omitempty,oneof=success failed rolled_back"`
}

type CreateAlertRequest struct {
	ServiceID   string `json:"service_id" binding:"omitempty,uuid"`
	Title       string `json:"title" binding:"required,max=200"`
	Type        string `json:"type" binding:"required,oneof=high_latency high_error_rate service_down"`
	Severity    string `json:"severity" binding:"required,oneof=low medium high critical"`
	Description string `json:"description" binding:"max=500"`
}

type AlertRuleRequest struct {
	Name       string  `json:"name" binding:"required,max=200"`
	ServiceID  string  `json:"service_id" binding:"omitempty,uuid"`
	Metric     string  `json:"metric" binding:"required,max=50"`
	Operator   string  `json:"operator" binding:"required,oneof=gt lt gte lte"`
	Threshold  float64 `json:"threshold"`
	ForSeconds int     `json:"for_seconds" binding:"min=0,max=86400"`
	Severity   string  `json:"severity" binding:"required,oneof=low medium high critical"`
	Type       string  `json:"type" binding:"required,oneof=high_latency high_error_rate service_down"`
	NotifyType      string `json:"notify_type" binding:"omitempty,oneof=none slack telegram webhook"`
	NotifyURL       string `json:"notify_url" binding:"omitempty,url,max=500"`
	NotifyChannelID string `json:"notify_channel_id" binding:"omitempty,uuid"`
	Enabled         *bool  `json:"enabled"`
}

// Ingestion DTOs (used by collector / agents).

type IngestMetricsRequest struct {
	Points []struct {
		ServiceID   string  `json:"service_id" binding:"required,uuid"`
		ServiceName string  `json:"service_name"`
		MetricName  string  `json:"metric_name" binding:"required"`
		Value       float64 `json:"value"`
		Timestamp   string  `json:"timestamp"`
	} `json:"points" binding:"required,dive"`
}

type IngestLogsRequest struct {
	Logs []struct {
		ServiceID   string `json:"service_id" binding:"required,uuid"`
		ServiceName string `json:"service_name"`
		Level       string `json:"level" binding:"required,oneof=info warning error"`
		Message     string `json:"message" binding:"required"`
		Metadata    string `json:"metadata"`
		Timestamp   string `json:"timestamp"`
	} `json:"logs" binding:"required,dive"`
}
