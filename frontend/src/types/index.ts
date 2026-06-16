export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export type OrgRole = "owner" | "admin" | "member";

export interface Organization {
  id: string;
  name: string;
  slug: string;
  created_by: string;
  role?: OrgRole;
  created_at: string;
}

export interface TeamMember {
  id: string;
  organization_id: string;
  user_id: string;
  email: string;
  name: string;
  role: OrgRole;
  created_at: string;
}

export interface Project {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
}

export type ServiceStatus = "healthy" | "degraded" | "down";

export interface Service {
  id: string;
  project_id: string;
  name: string;
  environment: string;
  status: ServiceStatus;
  created_at: string;
}

export interface Deployment {
  id: string;
  project_id: string;
  service_id: string;
  service_name: string;
  version: string;
  commit_sha: string;
  environment: string;
  deployed_by: string;
  status: string;
  created_at: string;
}

export type AlertSeverity = "low" | "medium" | "high" | "critical";
export type AlertStatus = "active" | "resolved";
export type AlertType = "high_latency" | "high_error_rate" | "service_down";

export interface Alert {
  id: string;
  project_id: string;
  service_id: string | null;
  service_name?: string;
  title: string;
  type: AlertType;
  severity: AlertSeverity;
  status: AlertStatus;
  description: string;
  created_at: string;
  resolved_at: string | null;
}

export type TimelineEventType =
  | "deployment"
  | "alert"
  | "metric_spike"
  | "error_spike"
  | "recovery";

export interface TimelineEvent {
  id: string;
  project_id: string;
  service_id: string | null;
  service_name?: string;
  type: TimelineEventType;
  title: string;
  description: string;
  severity: AlertSeverity | null;
  occurred_at: string;
}

export interface SeriesPoint {
  timestamp: string;
  value: number;
}

export interface MetricSeries {
  metric_name: string;
  service_id: string;
  points: SeriesPoint[];
}

export interface LogEntry {
  service_id: string;
  service_name: string;
  level: "info" | "warning" | "error";
  message: string;
  metadata: string;
  timestamp: string;
}

export interface DashboardSummary {
  system_health: ServiceStatus;
  active_alerts: number;
  deployments_today: number;
  error_rate: number;
}

export interface RCAResult {
  summary: string;
  confidence: number;
  evidence: string[];
  findings: string[];
}

export interface IngestKey {
  id: string;
  project_id: string;
  name: string;
  prefix: string;
  token?: string; // present only in the create response
  created_at: string;
  last_used_at: string | null;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  user: User;
}

export const METRIC_NAMES = [
  "cpu_usage",
  "memory_usage",
  "request_count",
  "request_rate",
  "error_rate",
  "latency_p50",
  "latency_p95",
  "latency_p99",
] as const;

export type MetricName = (typeof METRIC_NAMES)[number];
