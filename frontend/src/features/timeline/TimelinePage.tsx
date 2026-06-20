import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Sparkles, RefreshCw } from "lucide-react";
import { api } from "@/services/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Select,
} from "@/components/ui/primitives";
import { PageHeader, Spinner, TimeRangeControl } from "@/components/common";
import { TimelineFeed, EVENT_META } from "./TimelineFeed";
import { rangeWindow, useRangeStore } from "@/store/range";
import { cn } from "@/lib/utils";
import type { TimelineEventType } from "@/types";

const TYPE_ORDER: TimelineEventType[] = ["deployment", "alert", "error_spike", "metric_spike", "recovery"];

export function TimelinePage() {
  const { projectId } = useParams();
  const qc = useQueryClient();
  const { range, live } = useRangeStore();
  const [hidden, setHidden] = useState<Set<TimelineEventType>>(new Set());
  const [service, setService] = useState("");

  const timeline = useQuery({
    queryKey: ["timeline", projectId, range],
    queryFn: () => {
      const w = rangeWindow(range);
      return api.timeline(projectId!, w.from, w.to);
    },
    refetchInterval: live ? 15000 : false,
  });
  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });
  const rca = useQuery({
    queryKey: ["rca", projectId, range],
    queryFn: () => api.analyze(projectId!, rangeWindow(range).minutes),
  });

  useWebSocket(projectId, () => {
    qc.invalidateQueries({ queryKey: ["timeline", projectId] });
  });

  const events = timeline.data ?? [];
  const counts = TYPE_ORDER.reduce<Record<string, number>>((acc, t) => {
    acc[t] = events.filter((e) => e.type === t).length;
    return acc;
  }, {});
  const filtered = events.filter(
    (e) => !hidden.has(e.type) && (!service || e.service_id === service)
  );

  const toggle = (t: TimelineEventType) =>
    setHidden((prev) => {
      const next = new Set(prev);
      if (next.has(t)) next.delete(t);
      else next.add(t);
      return next;
    });

  return (
    <div>
      <PageHeader
        title="Incident Timeline"
        description="Chronological view of deployments, spikes, errors and alerts"
        actions={
          <div className="flex items-center gap-2">
            <Select value={service} onChange={(e) => setService(e.target.value)} className="w-40">
              <option value="">All services</option>
              {services.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
            <TimeRangeControl />
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                timeline.refetch();
                rca.refetch();
              }}
            >
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        }
      />

      {/* Root Cause Analysis */}
      <Card className="mb-6 border-primary/30 bg-gradient-to-br from-primary/5 to-transparent">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-primary" />
            Root Cause Analysis
          </CardTitle>
        </CardHeader>
        <CardContent>
          {rca.isLoading ? (
            <Spinner />
          ) : rca.data ? (
            <div>
              <div className="flex items-start justify-between gap-4">
                <p className="flex-1 text-sm leading-relaxed text-fg">{rca.data.summary}</p>
                <div className="shrink-0 text-right">
                  <div className="text-2xl font-semibold text-primary">
                    {Math.round(rca.data.confidence * 100)}%
                  </div>
                  <div className="text-xs text-fg-muted">confidence</div>
                </div>
              </div>
              {(rca.data.evidence?.length ?? 0) > 0 && (
                <div className="mt-4 border-t border-border pt-3">
                  <p className="mb-2 text-xs font-medium text-fg-muted">Evidence</p>
                  <ul className="space-y-1">
                    {(rca.data.evidence ?? []).map((e, i) => (
                      <li key={i} className="flex gap-2 text-xs text-fg-muted">
                        <span className="text-primary">→</span>
                        {e}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-fg-muted">Analysis unavailable.</p>
          )}
        </CardContent>
      </Card>

      {/* Type filter / summary chips */}
      <div className="mb-4 flex flex-wrap gap-2">
        {TYPE_ORDER.map((t) => {
          const m = EVENT_META[t];
          const on = !hidden.has(t);
          return (
            <button
              key={t}
              onClick={() => toggle(t)}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors",
                on ? "border-border bg-surface-2 text-fg" : "border-border text-fg-muted/60 hover:text-fg"
              )}
            >
              <span className={cn("h-2 w-2 rounded-full", m.ring, on ? "" : "opacity-40")} />
              {m.label}
              <span className="font-mono text-fg-muted">{counts[t] ?? 0}</span>
            </button>
          );
        })}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Event Sequence</CardTitle>
        </CardHeader>
        <CardContent>
          {timeline.isLoading ? <Spinner /> : <TimelineFeed events={filtered} />}
        </CardContent>
      </Card>
    </div>
  );
}
