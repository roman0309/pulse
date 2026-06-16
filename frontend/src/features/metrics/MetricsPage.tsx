import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useQueries } from "@tanstack/react-query";
import { api } from "@/services/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Select,
} from "@/components/ui/primitives";
import { PageHeader, Spinner } from "@/components/common";
import { MetricChart } from "@/components/common/MetricChart";
import { METRIC_NAMES } from "@/types";

const RANGES = [
  { label: "Last 30m", minutes: 30, step: 30 },
  { label: "Last 1h", minutes: 60, step: 60 },
  { label: "Last 3h", minutes: 180, step: 120 },
  { label: "Last 24h", minutes: 1440, step: 600 },
];

const METRIC_META: Record<string, { label: string; unit: string }> = {
  cpu_usage: { label: "CPU Usage", unit: "%" },
  memory_usage: { label: "Memory Usage", unit: "%" },
  request_count: { label: "Request Count", unit: "" },
  request_rate: { label: "Request Rate", unit: "/s" },
  error_rate: { label: "Error Rate", unit: "%" },
  latency_p50: { label: "Latency P50", unit: "ms" },
  latency_p95: { label: "Latency P95", unit: "ms" },
  latency_p99: { label: "Latency P99", unit: "ms" },
};

export function MetricsPage() {
  const { projectId } = useParams();
  const [serviceId, setServiceId] = useState("");
  const [rangeIdx, setRangeIdx] = useState(1);
  const [live, setLive] = useState(true);
  const range = RANGES[rangeIdx];

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });

  const from = new Date(Date.now() - range.minutes * 60000).toISOString();

  const results = useQueries({
    queries: METRIC_NAMES.map((metric) => ({
      queryKey: ["metric", projectId, metric, serviceId, rangeIdx],
      queryFn: () =>
        api.metrics(projectId!, metric, {
          serviceId: serviceId || undefined,
          from,
          step: range.step,
        }),
      refetchInterval: live ? 15000 : false,
    })),
  });

  const serviceNames = Object.fromEntries(
    (services.data ?? []).map((s) => [s.id, s.name])
  );

  return (
    <div>
      <PageHeader
        title="Metrics"
        description="Live time-series across all your services"
        actions={
          <div className="flex items-center gap-2">
            <Select
              value={serviceId}
              onChange={(e) => setServiceId(e.target.value)}
            >
              <option value="">All services</option>
              {services.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
            <Select
              value={rangeIdx}
              onChange={(e) => setRangeIdx(Number(e.target.value))}
            >
              {RANGES.map((r, i) => (
                <option key={i} value={i}>
                  {r.label}
                </option>
              ))}
            </Select>
            <button
              onClick={() => setLive((v) => !v)}
              className={`flex items-center gap-1.5 rounded-md border px-3 h-9 text-sm transition ${
                live
                  ? "border-success/40 text-success"
                  : "border-border text-fg-muted"
              }`}
            >
              <span
                className={`h-2 w-2 rounded-full ${live ? "bg-success animate-pulse" : "bg-fg-muted"}`}
              />
              Live
            </button>
          </div>
        }
      />

      <div className="grid md:grid-cols-2 gap-4">
        {METRIC_NAMES.map((metric, i) => {
          const meta = METRIC_META[metric];
          const q = results[i];
          return (
            <Card key={metric}>
              <CardHeader>
                <CardTitle>{meta.label}</CardTitle>
              </CardHeader>
              <CardContent>
                {q.isLoading ? (
                  <div className="h-[200px] flex items-center justify-center">
                    <Spinner />
                  </div>
                ) : (
                  <MetricChart
                    series={q.data ?? []}
                    serviceNames={serviceNames}
                    unit={meta.unit}
                    height={200}
                  />
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
