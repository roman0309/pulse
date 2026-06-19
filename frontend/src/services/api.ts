import { useAuthStore } from "@/store/auth";
import type {
  Alert,
  DashboardSummary,
  AlertRule,
  Deployment,
  IngestKey,
  LogEntry,
  ManagedServer,
  MetricSeries,
  Organization,
  Project,
  RCAResult,
  Service,
  TeamMember,
  TimelineEvent,
  TokenPair,
  User,
} from "@/types";

const BASE = "/api/v1";

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

let refreshing: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();
  if (!refreshToken) return false;
  if (!refreshing) {
    refreshing = fetch(`${BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
      .then(async (res) => {
        if (!res.ok) {
          logout();
          return false;
        }
        const data: TokenPair = await res.json();
        setTokens(data.access_token, data.refresh_token);
        return true;
      })
      .catch(() => {
        logout();
        return false;
      })
      .finally(() => {
        refreshing = null;
      });
  }
  return refreshing;
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  retry = true
): Promise<T> {
  const { accessToken } = useAuthStore.getState();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  if (res.status === 401 && retry) {
    const ok = await tryRefresh();
    if (ok) return request<T>(path, options, false);
  }

  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = await res.json();
      msg = body.error || msg;
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  // --- auth ---
  register: (email: string, name: string, password: string) =>
    request<TokenPair>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, name, password }),
    }),
  login: (email: string, password: string) =>
    request<TokenPair>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  logout: (refresh_token: string) =>
    request("/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token }),
    }),
  me: () => request<User>("/auth/me"),

  // --- runtime config ---
  meta: () => request<{ ingest_url: string; agent_image: string }>("/meta"),

  // --- organizations ---
  listOrgs: () =>
    request<{ organizations: Organization[] }>("/organizations").then(
      (r) => r.organizations ?? []
    ),
  createOrg: (name: string) =>
    request<Organization>("/organizations", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  listMembers: (orgId: string) =>
    request<{ members: TeamMember[] }>(
      `/organizations/${orgId}/members`
    ).then((r) => r.members ?? []),

  // --- projects ---
  listProjects: (orgId: string) =>
    request<{ projects: Project[] }>(
      `/organizations/${orgId}/projects`
    ).then((r) => r.projects ?? []),
  createProject: (orgId: string, name: string, description: string) =>
    request<Project>(`/organizations/${orgId}/projects`, {
      method: "POST",
      body: JSON.stringify({ name, description }),
    }),
  getProject: (projectId: string) =>
    request<Project>(`/projects/${projectId}`),
  updateProject: (projectId: string, name: string, description: string) =>
    request<Project>(`/projects/${projectId}`, {
      method: "PUT",
      body: JSON.stringify({ name, description }),
    }),
  deleteProject: (projectId: string) =>
    request(`/projects/${projectId}`, { method: "DELETE" }),

  // --- dashboard / timeline / rca ---
  dashboard: (projectId: string) =>
    request<DashboardSummary>(`/projects/${projectId}/dashboard`),
  timeline: (projectId: string, from?: string, to?: string) =>
    request<{ events: TimelineEvent[] }>(
      `/projects/${projectId}/timeline${qs({ from, to })}`
    ).then((r) => r.events ?? []),
  analyze: (projectId: string, windowMinutes = 120) =>
    request<RCAResult>(
      `/projects/${projectId}/analyze?window_minutes=${windowMinutes}`
    ),

  // --- services ---
  listServices: (projectId: string) =>
    request<{ services: Service[] }>(
      `/projects/${projectId}/services`
    ).then((r) => r.services ?? []),
  createService: (
    projectId: string,
    body: { name: string; environment: string; status?: string }
  ) =>
    request<Service>(`/projects/${projectId}/services`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateService: (
    serviceId: string,
    body: { name: string; environment: string; status: string }
  ) =>
    request<Service>(`/services/${serviceId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deleteService: (serviceId: string) =>
    request(`/services/${serviceId}`, { method: "DELETE" }),

  // --- metrics ---
  metrics: (
    projectId: string,
    metric: string,
    opts: { serviceId?: string; from?: string; to?: string; step?: number } = {}
  ) =>
    request<{ series: MetricSeries[] }>(
      `/projects/${projectId}/metrics${qs({
        metric,
        service_id: opts.serviceId,
        from: opts.from,
        to: opts.to,
        step: opts.step,
      })}`
    ).then((r) => r.series ?? []),

  // --- logs ---
  logs: (
    projectId: string,
    opts: {
      serviceId?: string;
      level?: string;
      search?: string;
      limit?: number;
      offset?: number;
    } = {}
  ) =>
    request<{ logs: LogEntry[] }>(
      `/projects/${projectId}/logs${qs({
        service_id: opts.serviceId,
        level: opts.level,
        search: opts.search,
        limit: opts.limit,
        offset: opts.offset,
      })}`
    ).then((r) => r.logs ?? []),

  // --- deployments ---
  deployments: (projectId: string, serviceId?: string) =>
    request<{ deployments: Deployment[] }>(
      `/projects/${projectId}/deployments${qs({ service_id: serviceId })}`
    ).then((r) => r.deployments ?? []),
  createDeployment: (
    projectId: string,
    body: {
      service_id: string;
      version: string;
      commit_sha?: string;
      environment?: string;
      deployed_by?: string;
    }
  ) =>
    request<Deployment>(`/projects/${projectId}/deployments`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  // --- alerts ---
  alerts: (projectId: string, status?: string) =>
    request<{ alerts: Alert[] }>(
      `/projects/${projectId}/alerts${qs({ status })}`
    ).then((r) => r.alerts ?? []),
  createAlert: (
    projectId: string,
    body: {
      service_id?: string;
      title: string;
      type: string;
      severity: string;
      description?: string;
    }
  ) =>
    request<Alert>(`/projects/${projectId}/alerts`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  resolveAlert: (alertId: string) =>
    request<Alert>(`/alerts/${alertId}/resolve`, { method: "POST" }),

  // --- alert rules (alerting engine) ---
  listAlertRules: (projectId: string) =>
    request<{ rules: AlertRule[] }>(
      `/projects/${projectId}/alert-rules`
    ).then((r) => r.rules ?? []),
  createAlertRule: (projectId: string, body: Partial<AlertRule>) =>
    request<AlertRule>(`/projects/${projectId}/alert-rules`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateAlertRule: (projectId: string, ruleId: string, body: Partial<AlertRule>) =>
    request<AlertRule>(`/projects/${projectId}/alert-rules/${ruleId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deleteAlertRule: (projectId: string, ruleId: string) =>
    request(`/projects/${projectId}/alert-rules/${ruleId}`, { method: "DELETE" }),

  // --- managed servers (remote agent management) ---
  listServers: (projectId: string) =>
    request<{ servers: ManagedServer[] }>(
      `/projects/${projectId}/servers`
    ).then((r) => r.servers ?? []),
  addServer: (
    projectId: string,
    body: { name: string; host: string; port: number; user: string; auth_method: string; secret: string }
  ) =>
    request<ManagedServer>(`/projects/${projectId}/servers`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  deleteServer: (projectId: string, serverId: string) =>
    request(`/projects/${projectId}/servers/${serverId}`, { method: "DELETE" }),
  serverAction: (projectId: string, serverId: string, action: "install" | "remove" | "status") =>
    request<ManagedServer>(`/projects/${projectId}/servers/${serverId}/${action}`, {
      method: "POST",
    }),
  installBeyla: (projectId: string, serverId: string, ports: string) =>
    request<ManagedServer>(`/projects/${projectId}/servers/${serverId}/beyla`, {
      method: "POST",
      body: JSON.stringify({ ports }),
    }),
  runServerCommand: (projectId: string, serverId: string, command: string) =>
    request<ManagedServer>(`/projects/${projectId}/servers/${serverId}/run`, {
      method: "POST",
      body: JSON.stringify({ command }),
    }),

  // --- agent control channel (live, outbound WS) ---
  listAgents: (projectId: string) =>
    request<{ agents: string[] }>(`/projects/${projectId}/agents`).then(
      (r) => r.agents ?? []
    ),
  sendAgentCommand: (
    projectId: string,
    agentId: string,
    cmd: "ping" | "status" | "install_beyla" | "remove",
    args?: Record<string, string>
  ) =>
    request<{ id: string; ok: boolean; output: string }>(
      `/projects/${projectId}/agents/${encodeURIComponent(agentId)}/command`,
      { method: "POST", body: JSON.stringify({ cmd, args }) }
    ),

  // --- ingest keys (server onboarding) ---
  listIngestKeys: (projectId: string) =>
    request<{ keys: IngestKey[] }>(
      `/projects/${projectId}/ingest-keys`
    ).then((r) => r.keys ?? []),
  createIngestKey: (projectId: string, name: string) =>
    request<IngestKey>(`/projects/${projectId}/ingest-keys`, {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  deleteIngestKey: (projectId: string, keyId: string) =>
    request(`/projects/${projectId}/ingest-keys/${keyId}`, {
      method: "DELETE",
    }),
};

function qs(params: Record<string, string | number | undefined>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== ""
  );
  if (entries.length === 0) return "";
  return "?" + entries.map(([k, v]) => `${k}=${encodeURIComponent(v!)}`).join("&");
}

export { ApiError };
