import { cn } from "@/lib/utils";
import { RANGE_KEYS, useRangeStore } from "@/store/range";

// Shared time-range + auto-refresh control. Drop into a PageHeader's actions.
export function TimeRangeControl({ live = true }: { live?: boolean }) {
  const { range, setRange, live: isLive, toggleLive } = useRangeStore();
  return (
    <div className="flex items-center gap-1.5">
      <div className="flex overflow-hidden rounded-md border border-border">
        {RANGE_KEYS.map((r) => (
          <button
            key={r}
            onClick={() => setRange(r)}
            className={cn(
              "px-2.5 py-1 text-xs transition-colors",
              range === r ? "bg-surface-2 text-fg" : "text-fg-muted hover:text-fg hover:bg-surface-2"
            )}
          >
            {r}
          </button>
        ))}
      </div>
      {live && (
        <button
          onClick={toggleLive}
          title={isLive ? "Auto-refresh on" : "Auto-refresh off"}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors",
            isLive ? "border-success/40 text-success" : "border-border text-fg-muted hover:text-fg"
          )}
        >
          <span className={cn("h-1.5 w-1.5 rounded-full", isLive ? "animate-pulse bg-success" : "bg-fg-muted")} />
          Live
        </button>
      )}
    </div>
  );
}
