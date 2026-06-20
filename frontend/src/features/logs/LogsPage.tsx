import { useEffect, useRef, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { api } from "@/services/api";
import { Card, Input, Select } from "@/components/ui/primitives";
import { PageHeader, Spinner, EmptyState, TimeRangeControl } from "@/components/common";
import { formatDateTime } from "@/lib/utils";
import { rangeWindow, useRangeStore } from "@/store/range";

const PAGE = 50;

export function LogsPage() {
  const { projectId } = useParams();
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [level, setLevel] = useState("");
  const [searchParams] = useSearchParams();
  const [serviceId, setServiceId] = useState(searchParams.get("service") ?? "");
  const { range } = useRangeStore();
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

  const logs = query.data?.pages.flat() ?? [];

  return (
    <div>
      <PageHeader
        title="Logs"
        description="Search and filter structured logs across services"
        actions={<TimeRangeControl live={false} />}
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
          <EmptyState title="No logs found" description="Try adjusting your filters." />
        ) : (
          <div className="divide-y divide-border font-mono text-xs">
            {logs.map((l, i) => (
              <div
                key={i}
                className="flex items-start gap-3 px-4 py-2.5 hover:bg-surface-2 transition"
              >
                <span className="text-fg-muted whitespace-nowrap">
                  {formatDateTime(l.timestamp)}
                </span>
                <span
                  className={`uppercase font-medium w-14 shrink-0 ${
                    l.level === "error"
                      ? "text-danger"
                      : l.level === "warning"
                        ? "text-warning"
                        : "text-success"
                  }`}
                >
                  {l.level}
                </span>
                <span className="text-primary whitespace-nowrap">
                  {l.service_name}
                </span>
                <span className="text-fg flex-1 break-all">{l.message}</span>
              </div>
            ))}
            <div ref={sentinel} className="h-10 flex items-center justify-center">
              {query.isFetchingNextPage && <Spinner />}
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
