import {
  GitCommitHorizontal,
  AlertTriangle,
  TrendingUp,
  Bug,
  CheckCircle2,
} from "lucide-react";
import type { TimelineEvent, TimelineEventType } from "@/types";
import { formatTime, relativeTime } from "@/lib/utils";
import { EmptyState, SeverityBadge } from "@/components/common";

export const EVENT_META: Record<
  TimelineEventType,
  { icon: typeof GitCommitHorizontal; color: string; ring: string; label: string }
> = {
  deployment: { icon: GitCommitHorizontal, color: "text-primary", ring: "bg-primary/15", label: "Deployments" },
  alert: { icon: AlertTriangle, color: "text-danger", ring: "bg-danger/15", label: "Alerts" },
  metric_spike: { icon: TrendingUp, color: "text-warning", ring: "bg-warning/15", label: "Spikes" },
  error_spike: { icon: Bug, color: "text-danger", ring: "bg-danger/15", label: "Errors" },
  recovery: { icon: CheckCircle2, color: "text-success", ring: "bg-success/15", label: "Recoveries" },
};

function dayLabel(iso: string) {
  const d = new Date(iso);
  const today = new Date();
  const yesterday = new Date(Date.now() - 86_400_000);
  const same = (a: Date, b: Date) => a.toDateString() === b.toDateString();
  if (same(d, today)) return "Today";
  if (same(d, yesterday)) return "Yesterday";
  return d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
}

export function TimelineFeed({ events }: { events: TimelineEvent[] }) {
  if (events.length === 0) {
    return (
      <EmptyState
        title="No events in this window"
        description="Deployments, metric spikes, errors and alerts will appear here as they happen."
      />
    );
  }

  // Group consecutive events by calendar day (events arrive newest-first).
  const groups: { day: string; items: TimelineEvent[] }[] = [];
  for (const ev of events) {
    const day = dayLabel(ev.occurred_at);
    const last = groups[groups.length - 1];
    if (last && last.day === day) last.items.push(ev);
    else groups.push({ day, items: [ev] });
  }

  return (
    <div className="space-y-5">
      {groups.map((g) => (
        <div key={g.day}>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">{g.day}</p>
          <ol className="relative">
            {g.items.map((ev, i) => {
              const c = EVENT_META[ev.type] ?? EVENT_META.metric_spike;
              const Icon = c.icon;
              const isLast = i === g.items.length - 1;
              return (
                <li key={ev.id} className="relative flex gap-4 pb-6">
                  {!isLast && <span className="absolute left-[15px] top-8 bottom-0 w-px bg-border" />}
                  <span className={`relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${c.ring}`}>
                    <Icon className={`h-4 w-4 ${c.color}`} />
                  </span>
                  <div className="-mt-0.5 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-fg-muted">{formatTime(ev.occurred_at)}</span>
                      <span className="text-sm font-medium text-fg">{ev.title}</span>
                      {ev.severity && <SeverityBadge severity={ev.severity} />}
                      {ev.service_name && <span className="text-xs text-fg-muted">· {ev.service_name}</span>}
                    </div>
                    {ev.description && <p className="mt-0.5 text-sm text-fg-muted">{ev.description}</p>}
                    <p className="mt-0.5 text-xs text-fg-muted/70">{relativeTime(ev.occurred_at)}</p>
                  </div>
                </li>
              );
            })}
          </ol>
        </div>
      ))}
    </div>
  );
}
