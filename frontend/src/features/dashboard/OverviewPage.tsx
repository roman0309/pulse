import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import { PageHeader, Spinner } from "@/components/common";
import { TimelineFeed } from "@/features/timeline/TimelineFeed";
import { MetricChart } from "@/components/common/MetricChart";
import { relativeTime } from "@/lib/utils";

export function OverviewPage() {
  const { projectId } = useParams();
  const qc = useQueryClient();

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
    queryKey: ["overview-latency", projectId],
    queryFn: () => api.metrics(projectId!, "latency_p95", { step: 120 }),
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
