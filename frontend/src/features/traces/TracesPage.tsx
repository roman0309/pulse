import { useMemo, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, Network, AlertTriangle, ScrollText } from "lucide-react";
import { api } from "@/services/api";
import { Badge, Card, CardContent, Select } from "@/components/ui/primitives";
import { PageHeader, Spinner, EmptyState, TimeRangeControl } from "@/components/common";
import { cn, relativeTime } from "@/lib/utils";
import { rangeWindow, useRangeStore } from "@/store/range";
import type { Span, TraceSummary } from "@/types";

const PALETTE = ["#6366f1", "#10b981", "#f59e0b", "#ec4899", "#06b6d4", "#a855f7", "#ef4444", "#84cc16"];
function serviceColor(name: string) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  return PALETTE[h % PALETTE.length];
}

function fmtMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
  if (ms >= 1) return `${ms.toFixed(1)}ms`;
  return `${(ms * 1000).toFixed(0)}µs`;
}

export function TracesPage() {
  const { projectId } = useParams();
  const [searchParams] = useSearchParams();
  const { range, live } = useRangeStore();
  const [selected, setSelected] = useState<string | null>(searchParams.get("trace"));
  const [service, setService] = useState("");
  const [errorsOnly, setErrorsOnly] = useState(false);

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });
  const traces = useQuery({
    queryKey: ["traces", projectId, range, service],
    queryFn: () =>
      api.listTraces(projectId!, { from: rangeWindow(range).from, limit: 100, service: service || undefined }),
    refetchInterval: live ? 15_000 : false,
    enabled: !selected,
  });

  if (selected) {
    return <TraceDetail projectId={projectId!} traceId={selected} onBack={() => setSelected(null)} />;
  }

  const list = (traces.data ?? []).filter((t) => !errorsOnly || t.error_count > 0);

  return (
    <div>
      <PageHeader
        title="Traces"
        description="Distributed request traces (zero-code via Beyla or any OpenTelemetry SDK)"
        actions={
          <div className="flex items-center gap-2">
            <Select value={service} onChange={(e) => setService(e.target.value)} className="w-40">
              <option value="">All services</option>
              {services.data?.map((s) => (
                <option key={s.id} value={s.name}>
                  {s.name}
                </option>
              ))}
            </Select>
            <button
              onClick={() => setErrorsOnly((v) => !v)}
              className={cn(
                "rounded-md border px-2.5 py-1 text-xs transition-colors",
                errorsOnly ? "border-danger/40 text-danger" : "border-border text-fg-muted hover:text-fg"
              )}
            >
              Errors only
            </button>
            <TimeRangeControl />
          </div>
        }
      />

      {traces.isLoading ? (
        <Spinner />
      ) : list.length > 0 ? (
        <Card>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {list.map((t) => (
                <TraceRow key={t.trace_id} trace={t} onClick={() => setSelected(t.trace_id)} />
              ))}
            </div>
          </CardContent>
        </Card>
      ) : (
        <EmptyState
          icon={<Network className="h-7 w-7" />}
          title="No traces yet"
          description="Install app metrics (Beyla) on a server, generate some traffic, and traces will appear here."
        />
      )}
    </div>
  );
}

function TraceRow({ trace, onClick }: { trace: TraceSummary; onClick: () => void }) {
  return (
    <button onClick={onClick} className="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-surface-2">
      <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ background: serviceColor(trace.root_service) }} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium text-fg">{trace.root_name || "(root)"}</span>
          <span className="truncate text-xs text-fg-muted">{trace.root_service}</span>
          {trace.error_count > 0 && (
            <Badge tone="danger">
              {trace.error_count} error{trace.error_count > 1 ? "s" : ""}
            </Badge>
          )}
        </div>
        <p className="text-xs text-fg-muted">
          {trace.span_count} span{trace.span_count > 1 ? "s" : ""} · {relativeTime(trace.start_time)}
        </p>
      </div>
      <span className="shrink-0 font-mono text-sm text-fg">{fmtMs(trace.duration_ms)}</span>
    </button>
  );
}

function TraceDetail({ projectId, traceId, onBack }: { projectId: string; traceId: string; onBack: () => void }) {
  const trace = useQuery({
    queryKey: ["trace", projectId, traceId],
    queryFn: () => api.getTrace(projectId, traceId),
  });

  return (
    <div>
      <div className="mb-3 flex items-center justify-between gap-2">
        <button onClick={onBack} className="inline-flex items-center gap-1 text-sm text-fg-muted transition hover:text-fg">
          <ChevronLeft className="h-4 w-4" /> Back to traces
        </button>
        <Link
          to={`/projects/${projectId}/logs?trace=${traceId}`}
          className="inline-flex items-center gap-1 text-sm text-primary transition hover:underline"
        >
          <ScrollText className="h-4 w-4" /> View logs
        </Link>
      </div>
      {trace.isLoading ? (
        <Spinner />
      ) : trace.data && trace.data.length > 0 ? (
        <Waterfall spans={trace.data} />
      ) : (
        <EmptyState icon={<Network className="h-7 w-7" />} title="Trace not found" description="It may have expired (traces are kept 7 days)." />
      )}
    </div>
  );
}

function Waterfall({ spans }: { spans: Span[] }) {
  const { ordered, depthOf, traceStart, total } = useMemo(() => layout(spans), [spans]);
  const [sel, setSel] = useState<Span | null>(null);
  const ticks = [0, 0.25, 0.5, 0.75, 1];

  return (
    <div className="space-y-3">
      <Card>
        <CardContent className="p-0">
          <div className="flex items-center justify-between border-b border-border px-4 py-2.5 text-xs text-fg-muted">
            <span>{spans.length} spans</span>
            <span className="font-mono">{fmtMs(total)} total</span>
          </div>

          {/* Time ruler */}
          <div className="flex items-center gap-3 px-4 pt-2">
            <div className="w-1/3" />
            <div className="relative h-4 flex-1">
              {ticks.map((t) => (
                <span
                  key={t}
                  className="absolute top-0 -translate-x-1/2 text-[10px] text-fg-muted first:translate-x-0 last:-translate-x-full"
                  style={{ left: `${t * 100}%` }}
                >
                  {fmtMs(total * t)}
                </span>
              ))}
            </div>
          </div>

          <div className="divide-y divide-border/60 pb-1">
            {ordered.map((s) => {
              const leftMs = new Date(s.start_time).getTime() - traceStart;
              const left = total > 0 ? (leftMs / total) * 100 : 0;
              const width = total > 0 ? Math.max((s.duration_ms / total) * 100, 0.5) : 0.5;
              const isErr = s.status_code === "error";
              const color = isErr ? "var(--color-danger, #ef4444)" : serviceColor(s.service_name);
              return (
                <button
                  key={s.span_id}
                  onClick={() => setSel(s)}
                  className={cn(
                    "flex w-full items-center gap-3 px-4 py-1.5 text-left transition hover:bg-surface-2",
                    sel?.span_id === s.span_id && "bg-surface-2"
                  )}
                >
                  <div className="w-1/3 min-w-0" style={{ paddingLeft: depthOf(s) * 14 }}>
                    <div className="flex items-center gap-1.5">
                      {isErr && <AlertTriangle className="h-3 w-3 shrink-0 text-danger" />}
                      <span className="truncate text-xs font-medium text-fg" title={s.name}>
                        {s.name}
                      </span>
                    </div>
                    <span className="truncate text-[11px] text-fg-muted">{s.service_name}</span>
                  </div>
                  <div className="relative h-5 flex-1 rounded bg-surface-2">
                    <div className="absolute top-0 h-full rounded" style={{ left: `${left}%`, width: `${width}%`, background: color }} />
                    <span
                      className="absolute top-1/2 -translate-y-1/2 whitespace-nowrap px-1 text-[10px] text-fg-muted"
                      style={{ left: `clamp(0%, ${left}%, 88%)` }}
                    >
                      {fmtMs(s.duration_ms)}
                    </span>
                  </div>
                </button>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {sel && <SpanDetail span={sel} traceStart={traceStart} onClose={() => setSel(null)} />}
    </div>
  );
}

function SpanDetail({ span, traceStart, onClose }: { span: Span; traceStart: number; onClose: () => void }) {
  let attrs: Record<string, unknown> = {};
  try {
    attrs = JSON.parse(span.attributes || "{}");
  } catch {
    attrs = {};
  }
  const offset = new Date(span.start_time).getTime() - traceStart;
  const rows: [string, string][] = [
    ["Service", span.service_name],
    ["Kind", span.kind],
    ["Status", span.status_code],
    ["Duration", fmtMs(span.duration_ms)],
    ["Start offset", `+${fmtMs(offset)}`],
    ["Span ID", span.span_id],
    ["Parent ID", span.parent_id || "—"],
  ];
  return (
    <Card>
      <CardContent className="p-4">
        <div className="mb-2 flex items-start justify-between gap-2">
          <p className="break-all text-sm font-medium text-fg">{span.name}</p>
          <button onClick={onClose} className="shrink-0 text-fg-muted transition hover:text-fg">
            <ChevronLeft className="h-4 w-4 rotate-90" />
          </button>
        </div>
        <div className="grid grid-cols-1 gap-x-6 gap-y-1 sm:grid-cols-2">
          {rows.map(([k, v]) => (
            <div key={k} className="flex justify-between gap-3 border-b border-border/40 py-1 text-xs">
              <span className="text-fg-muted">{k}</span>
              <span className="break-all text-right font-mono text-fg">{v}</span>
            </div>
          ))}
        </div>
        {Object.keys(attrs).length > 0 && (
          <div className="mt-3">
            <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-fg-muted">Attributes</p>
            <div className="grid grid-cols-1 gap-x-6 gap-y-0.5 sm:grid-cols-2">
              {Object.entries(attrs).map(([k, v]) => (
                <div key={k} className="flex justify-between gap-3 py-0.5 text-xs">
                  <span className="break-all text-fg-muted">{k}</span>
                  <span className="break-all text-right font-mono text-fg">{String(v)}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// layout computes DFS order (parent before children) and per-span depth.
function layout(spans: Span[]) {
  const byId = new Map(spans.map((s) => [s.span_id, s]));
  const children = new Map<string, Span[]>();
  const roots: Span[] = [];
  for (const s of spans) {
    if (s.parent_id && byId.has(s.parent_id)) {
      const arr = children.get(s.parent_id) ?? [];
      arr.push(s);
      children.set(s.parent_id, arr);
    } else {
      roots.push(s);
    }
  }
  const byStart = (a: Span, b: Span) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime();
  roots.sort(byStart);

  const ordered: Span[] = [];
  const depth = new Map<string, number>();
  const walk = (s: Span, d: number) => {
    depth.set(s.span_id, d);
    ordered.push(s);
    (children.get(s.span_id) ?? []).sort(byStart).forEach((c) => walk(c, d + 1));
  };
  roots.forEach((r) => walk(r, 0));

  let traceStart = Infinity;
  let traceEnd = -Infinity;
  for (const s of spans) {
    const start = new Date(s.start_time).getTime();
    traceStart = Math.min(traceStart, start);
    traceEnd = Math.max(traceEnd, start + s.duration_ms);
  }
  return {
    ordered,
    depthOf: (s: Span) => depth.get(s.span_id) ?? 0,
    traceStart,
    total: traceEnd - traceStart,
  };
}
