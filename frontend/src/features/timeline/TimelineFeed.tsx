import {
  GitCommitHorizontal,
  AlertTriangle,
  TrendingUp,
  Bug,
  CheckCircle2,
} from "lucide-react";
import type { TimelineEvent, TimelineEventType } from "@/types";
import { formatTime, relativeTime } from "@/lib/utils";
import { EmptyState } from "@/components/common";

const config: Record<
  TimelineEventType,
  { icon: typeof GitCommitHorizontal; color: string; ring: string }
> = {
  deployment: {
    icon: GitCommitHorizontal,
    color: "text-primary",
    ring: "bg-primary/15",
  },
  alert: { icon: AlertTriangle, color: "text-danger", ring: "bg-danger/15" },
  metric_spike: {
    icon: TrendingUp,
    color: "text-warning",
    ring: "bg-warning/15",
  },
  error_spike: { icon: Bug, color: "text-danger", ring: "bg-danger/15" },
  recovery: {
    icon: CheckCircle2,
    color: "text-success",
    ring: "bg-success/15",
  },
};

export function TimelineFeed({ events }: { events: TimelineEvent[] }) {
  if (events.length === 0) {
    return (
      <EmptyState
        title="No events in this window"
        description="Deployments, metric spikes, errors and alerts will appear here as they happen."
      />
    );
  }

  return (
    <ol className="relative">
      {events.map((ev, i) => {
        const c = config[ev.type] ?? config.metric_spike;
        const Icon = c.icon;
        const isLast = i === events.length - 1;
        return (
          <li key={ev.id} className="relative flex gap-4 pb-6">
            {!isLast && (
              <span className="absolute left-[15px] top-8 bottom-0 w-px bg-border" />
            )}
            <span
              className={`relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${c.ring}`}
            >
              <Icon className={`h-4 w-4 ${c.color}`} />
            </span>
            <div className="flex-1 -mt-0.5">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-mono text-xs text-fg-muted">
                  {formatTime(ev.occurred_at)}
                </span>
                <span className="text-sm font-medium text-fg">{ev.title}</span>
                {ev.service_name && (
                  <span className="text-xs text-fg-muted">
                    · {ev.service_name}
                  </span>
                )}
              </div>
              {ev.description && (
                <p className="text-sm text-fg-muted mt-0.5">
                  {ev.description}
                </p>
              )}
              <p className="text-xs text-fg-muted/70 mt-0.5">
                {relativeTime(ev.occurred_at)}
              </p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
