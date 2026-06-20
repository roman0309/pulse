import { Link, useParams } from "react-router-dom";
import { useQuery, useQueries, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  GitCommitHorizontal,
  TrendingUp,
} from "lucide-react";
import { api } from "@/services/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/primitives";
import { PageHeader, Spinner, TimeRangeControl, ServiceStatusDot } from "@/components/common";
import { TimelineFeed } from "@/features/timeline/TimelineFeed";
import { MetricChart } from "@/components/common/MetricChart";
import { rangeWindow, useRangeStore } from "@/store/range";
import { relativeTime } from "@/lib/utils";
import type { Service } from "@/types";

export function OverviewPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const { range, live } = useRangeStore();

  const dashboard = useQuery({
    queryKey: ["dashboard", projectId],
    queryFn: () => api.dashboard(projectId!),
    refetchInterval: 15000,
  });
  const timeline = useQuery({
    queryKey: ["timeline", projectId],
    queryFn: () => api.timeline(projectId!),
  });
  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });
  const latency = useQuery({
    queryKey: ["overview-latency", projectId, range],
    queryFn: () => {
      const w = rangeWindow(range);
      return api.metrics(projectId!, "latency_p95", { from: w.from, step: w.step });
    },
    refetchInterval: live ? 15000 : false,
  });
  const logs = useQuery({
    queryKey: ["overview-logs", projectId],
    queryFn: () => api.logs(projectId!, { limit: 6 }),
  });

  // realtime: refresh dashboard + timeline on any event
  useWebSocket(projectId, () => {
    qc.invalidateQueries({ queryKey: ["dashboard", projectId] });
    qc.invalidateQueries({ queryKey: ["timeline", projectId] });
  });

  const serviceNames = Object.fromEntries(
    (services.data ?? []).map((s) => [s.id, s.name])
  );

  const d = dashboard.data;
  const healthTone =
    d?.system_health === "healthy"
      ? "text-success"
      : d?.system_health === "degraded"
        ? "text-warning"
        : "text-danger";

  return (
    <div>
      <PageHeader
        title="Overview"
        description="System health and recent activity at a glance"
        actions={<TimeRangeControl />}
      />

      {/* Top stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <Stat
          icon={<Activity className="h-4 w-4" />}
          label="System Health"
          value={
            <span className={`capitalize ${healthTone}`}>
              {d?.system_health ?? "—"}
            </span>
          }
        />
        <Stat
          icon={<AlertTriangle className="h-4 w-4" />}
          label="Active Alerts"
          value={d?.active_alerts ?? "—"}
        />
        <Stat
          icon={<GitCommitHorizontal className="h-4 w-4" />}
          label="Deployments Today"
          value={d?.deployments_today ?? "—"}
        />
        <Stat
          icon={<TrendingUp className="h-4 w-4" />}
          label="Error Rate"
          value={d ? `${d.error_rate.toFixed(1)}%` : "—"}
        />
      </div>

      {/* Services at a glance */}
      {(services.data?.length ?? 0) > 0 && (
        <ServiceGrid projectId={projectId!} services={services.data!} />
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        {/* Timeline (center, most important) */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Incident Timeline</CardTitle>
          </CardHeader>
          <CardContent>
            {timeline.isLoading ? (
              <Spinner />
            ) : (
              <TimelineFeed events={timeline.data ?? []} />
            )}
          </CardContent>
        </Card>

        {/* Right column */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Latency P95</CardTitle>
            </CardHeader>
            <CardContent>
              {latency.isLoading ? (
                <Spinner />
              ) : (
                <MetricChart
                  series={latency.data ?? []}
                  serviceNames={serviceNames}
                  unit="ms"
                  height={180}
                />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Recent Logs</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {(logs.data ?? []).map((l, i) => (
                <div key={i} className="text-xs">
                  <span
                    className={
                      l.level === "error"
                        ? "text-danger"
                        : l.level === "warning"
                          ? "text-warning"
                          : "text-fg-muted"
                    }
                  >
                    [{l.level}]
                  </span>{" "}
                  <span className="text-fg-muted">{l.service_name}</span>{" "}
                  <span className="text-fg">{l.message}</span>
                </div>
              ))}
              {logs.data?.length === 0 && (
                <p className="text-xs text-fg-muted">No recent logs</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

function Stat({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <Card>
      <CardContent className="pt-5">
        <div className="flex items-center gap-2 text-fg-muted text-xs">
          {icon}
          {label}
        </div>
        <p className="text-2xl font-semibold text-fg mt-2">{value}</p>
      </CardContent>
    </Card>
  );
}

function ServiceGrid({ projectId, services }: { projectId: string; services: Service[] }) {
  const { range, live } = useRangeStore();
  const results = useQueries({
    queries: services.map((s) => ({
      queryKey: ["svc-latency", projectId, s.id, range],
      queryFn: () => {
        const w = rangeWindow(range);
        return api.metrics(projectId, "latency_p95", { serviceId: s.id, from: w.from, step: w.step });
      },
      refetchInterval: live ? 30000 : false,
    })),
  });

  return (
    <div className="mb-6">
      <h2 className="mb-2 text-sm font-semibold text-fg">Services</h2>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {services.map((s, i) => {
          const points = results[i].data?.[0]?.points ?? [];
          const last = points.length ? points[points.length - 1].value : null;
          return (
            <Card key={s.id} className="p-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-fg">{s.name}</p>
                  <p className="text-[11px] text-fg-muted">{s.environment}</p>
                </div>
                <ServiceStatusDot status={s.status} />
              </div>
              <div className="mt-2 flex items-end gap-2">
                <Sparkline points={points} />
                <span className="shrink-0 font-mono text-xs text-fg-muted">
                  {last != null ? `${last.toFixed(0)}ms` : "—"}
                </span>
              </div>
              <div className="mt-2 flex gap-3 text-[11px]">
                <Link to={`/projects/${projectId}/metrics?service=${s.id}`} className="text-primary hover:underline">
                  Metrics
                </Link>
                <Link to={`/projects/${projectId}/logs?service=${s.id}`} className="text-primary hover:underline">
                  Logs
                </Link>
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
}

function Sparkline({ points }: { points: { value: number }[] }) {
  if (points.length < 2) return <div className="h-8 flex-1" />;
  const vals = points.map((p) => p.value);
  const min = Math.min(...vals);
  const span = Math.max(...vals) - min || 1;
  const w = 120;
  const h = 32;
  const d = points
    .map((p, i) => {
      const x = (i / (points.length - 1)) * w;
      const y = h - ((p.value - min) / span) * h;
      return `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <svg viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" className="h-8 flex-1 text-primary">
      <path d={d} fill="none" stroke="currentColor" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
