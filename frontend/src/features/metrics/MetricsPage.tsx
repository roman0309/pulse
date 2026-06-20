import { useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useQuery, useQueries } from "@tanstack/react-query";
import { api } from "@/services/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Select,
} from "@/components/ui/primitives";
import { PageHeader, Spinner, TimeRangeControl } from "@/components/common";
import { MetricChart } from "@/components/common/MetricChart";
import { rangeWindow, useRangeStore } from "@/store/range";
import { METRIC_NAMES } from "@/types";

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
  const [searchParams] = useSearchParams();
  const [serviceId, setServiceId] = useState(searchParams.get("service") ?? "");
  const { range, live } = useRangeStore();

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });

  const results = useQueries({
    queries: METRIC_NAMES.map((metric) => ({
      queryKey: ["metric", projectId, metric, serviceId, range],
      queryFn: () => {
        const w = rangeWindow(range);
        return api.metrics(projectId!, metric, {
          serviceId: serviceId || undefined,
          from: w.from,
          step: w.step,
        });
      },
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
            <Select value={serviceId} onChange={(e) => setServiceId(e.target.value)}>
              <option value="">All services</option>
              {services.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
            <TimeRangeControl />
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
