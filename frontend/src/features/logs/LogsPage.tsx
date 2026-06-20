import { useEffect, useMemo, useRef, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Search, ScrollText, ChevronRight } from "lucide-react";
import { api } from "@/services/api";
import { Card, Input, Select } from "@/components/ui/primitives";
import { PageHeader, Spinner, EmptyState, TimeRangeControl } from "@/components/common";
import { cn, formatDateTime } from "@/lib/utils";
import { rangeWindow, useRangeStore } from "@/store/range";
import type { LogEntry } from "@/types";

const PAGE = 50;

const levelColor = (level: string) =>
  level === "error" ? "text-danger" : level === "warning" ? "text-warning" : "text-success";

const keyOf = (l: LogEntry) => `${l.timestamp}|${l.service_name}|${l.message}`;

export function LogsPage() {
  const { projectId } = useParams();
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [level, setLevel] = useState("");
  const [searchParams] = useSearchParams();
  const [serviceId, setServiceId] = useState(searchParams.get("service") ?? "");
  const { range } = useRangeStore();
  const [live, setLive] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [recent, setRecent] = useState<Set<string>>(new Set());
  const newestRef = useRef(0);
  const sentinel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  const services = useQuery({
    queryKey: ["services", projectId],
    queryFn: () => api.listServices(projectId!),
  });

  const query = useInfiniteQuery({
    queryKey: ["logs", projectId, debounced, level, serviceId, range],
    queryFn: ({ pageParam = 0 }) =>
      api.logs(projectId!, {
        search: debounced || undefined,
        level: level || undefined,
        serviceId: serviceId || undefined,
        from: rangeWindow(range).from,
        limit: PAGE,
        offset: pageParam,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) =>
      lastPage.length === PAGE ? pages.length * PAGE : undefined,
    refetchInterval: live ? 5000 : false,
  });

  // infinite scroll
  useEffect(() => {
    const el = sentinel.current;
    if (!el) return;
    const obs = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && query.hasNextPage && !query.isFetchingNextPage) {
        query.fetchNextPage();
      }
    });
    obs.observe(el);
    return () => obs.disconnect();
  }, [query]);

  const logs = useMemo(() => query.data?.pages.flat() ?? [], [query.data]);
  const topTs = logs[0]?.timestamp;

  // Highlight lines newer than the previously-seen newest, while live.
  useEffect(() => {
    if (!live || logs.length === 0) return;
    const prev = newestRef.current;
    const fresh = new Set<string>();
    for (const l of logs) {
      const t = new Date(l.timestamp).getTime();
      if (prev && t > prev) fresh.add(keyOf(l));
      else break;
    }
    newestRef.current = new Date(logs[0].timestamp).getTime();
    if (fresh.size > 0) {
      setRecent(fresh);
      const timer = setTimeout(() => setRecent(new Set()), 2500);
      return () => clearTimeout(timer);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [live, topTs]);

  const toggle = (k: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(k)) next.delete(k);
      else next.add(k);
      return next;
    });

  return (
    <div>
      <PageHeader
        title="Logs"
        description="Search and filter structured logs across services"
        actions={
          <div className="flex items-center gap-2">
            <button
              onClick={() => setLive((v) => !v)}
              title="Live tail — auto-refresh"
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors",
                live ? "border-success/40 text-success" : "border-border text-fg-muted hover:text-fg"
              )}
            >
              <span className={cn("h-1.5 w-1.5 rounded-full", live ? "animate-pulse bg-success" : "bg-fg-muted")} />
              Live
            </button>
            <TimeRangeControl live={false} />
          </div>
        }
      />

      <div className="flex flex-col sm:flex-row gap-2 mb-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-fg-muted" />
          <Input
            className="pl-9"
            placeholder="Full-text search…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Select value={level} onChange={(e) => setLevel(e.target.value)}>
          <option value="">All levels</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="error">Error</option>
        </Select>
        <Select value={serviceId} onChange={(e) => setServiceId(e.target.value)}>
          <option value="">All services</option>
          {services.data?.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
      </div>

      <Card className="overflow-hidden">
        {query.isLoading ? (
          <div className="p-8 flex justify-center">
            <Spinner />
          </div>
        ) : logs.length === 0 ? (
          <EmptyState
            icon={<ScrollText className="h-7 w-7" />}
            title="No logs in this window"
            description="The host agent ships your containers' logs automatically. Widen the time range or clear filters — or send app logs via OTLP (/otlp/v1/logs)."
          />
        ) : (
          <div className="divide-y divide-border text-xs">
            {logs.map((l, i) => {
              const k = keyOf(l);
              const open = expanded.has(k);
              return (
                <LogRow key={`${k}-${i}`} log={l} open={open} highlight={recent.has(k)} onToggle={() => toggle(k)} />
              );
            })}
            <div ref={sentinel} className="h-10 flex items-center justify-center">
              {query.isFetchingNextPage && <Spinner />}
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}

function LogRow({
  log,
  open,
  highlight,
  onToggle,
}: {
  log: LogEntry;
  open: boolean;
  highlight: boolean;
  onToggle: () => void;
}) {
  let meta: Record<string, unknown> = {};
  try {
    meta = JSON.parse(log.metadata || "{}");
  } catch {
    meta = {};
  }
  const metaEntries = Object.entries(meta);

  return (
    <div className={cn("transition-colors duration-1000", highlight && "bg-primary/10")}>
      <button onClick={onToggle} className="flex w-full items-start gap-3 px-4 py-2.5 text-left font-mono transition hover:bg-surface-2">
        <ChevronRight className={cn("mt-0.5 h-3.5 w-3.5 shrink-0 text-fg-muted transition-transform", open && "rotate-90")} />
        <span className="whitespace-nowrap text-fg-muted">{formatDateTime(log.timestamp)}</span>
        <span className={cn("w-14 shrink-0 font-medium uppercase", levelColor(log.level))}>{log.level}</span>
        <span className="whitespace-nowrap text-primary">{log.service_name}</span>
        <span className={cn("flex-1 text-fg", !open && "truncate")}>{log.message}</span>
      </button>
      {open && (
        <div className="space-y-2 border-t border-border/50 bg-surface-2/40 px-4 py-3 pl-10">
          <div>
            <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-fg-muted">Message</p>
            <pre className="whitespace-pre-wrap break-all font-mono text-xs text-fg">{log.message}</pre>
          </div>
          <div className="grid grid-cols-1 gap-x-6 sm:grid-cols-2">
            {[
              ["Service", log.service_name],
              ["Level", log.level],
              ["Time", formatDateTime(log.timestamp)],
            ].map(([key, val]) => (
              <div key={key} className="flex justify-between gap-3 border-b border-border/30 py-1">
                <span className="text-fg-muted">{key}</span>
                <span className="break-all text-right font-mono text-fg">{val}</span>
              </div>
            ))}
          </div>
          {metaEntries.length > 0 && (
            <div>
              <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-fg-muted">Metadata</p>
              <div className="grid grid-cols-1 gap-x-6 sm:grid-cols-2">
                {metaEntries.map(([key, val]) => (
                  <div key={key} className="flex justify-between gap-3 py-0.5">
                    <span className="break-all text-fg-muted">{key}</span>
                    <span className="break-all text-right font-mono text-fg">{String(val)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
