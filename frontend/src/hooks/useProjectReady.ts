import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";

// useProjectReady reports whether a project has started receiving telemetry.
// Until then the app keeps everything but the Connect page locked, so a new
// user is funnelled into connecting a server first.
//
// "Ready" = any ingest key has been used (data arrived) OR at least one
// service exists (auto-created on ingest, or added manually). This covers
// every ingestion path: agent, Beyla, OTLP, Prometheus and manual installs.
export function useProjectReady(projectId?: string) {
  const enabled = !!projectId;

  const keys = useQuery({
    queryKey: ["ingest-keys", projectId],
    queryFn: () => api.listIngestKeys(projectId!),
    enabled,
    refetchInterval: 20_000,
  });
  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
    enabled,
    refetchInterval: 20_000,
  });

  const ready =
    (keys.data?.some((k) => !!k.last_used_at) ?? false) ||
    ((services.data?.length ?? 0) > 0);

  // Only treat it as "still deciding" on the very first load, so the gate
  // doesn't flash while background refetches run.
  const loading =
    (keys.isLoading && keys.fetchStatus !== "idle") ||
    (services.isLoading && services.fetchStatus !== "idle");

  return { ready, loading };
}
